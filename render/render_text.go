package render

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"

	"github.com/clj/hrm-profile-tool/instructions"
)

type renderInstructionsTextOptions struct {
	showInstructionNumber bool
	showLineNumber        bool
	showRawInstruction    bool
	instructions          instructions.Instructions
}

// A RenderInstructionsText option
type RenderInstructionsTextOption func(*renderInstructionsTextOptions)

// Show instruction numbers (i.e. all decoded instructions are counted, including
// jump targets and comments)
func ShowInstructionNumbers() RenderInstructionsTextOption {
	return func(o *renderInstructionsTextOptions) {
		o.showInstructionNumber = true
	}
}

// Show line numbers as would be shown in the Human Resource Machine game
func ShowLineNumbers() RenderInstructionsTextOption {
	return func(o *renderInstructionsTextOptions) {
		o.showLineNumber = true
	}
}

// Show raw instruction (hex). Requires that the raw instruction data
// is passed using the RawInstructions option
func ShowRawInstructions() RenderInstructionsTextOption {
	return func(o *renderInstructionsTextOptions) {
		o.showRawInstruction = true
	}
}

// Raw instruction data for use with ShowRawInstructions. Using this option
// does *not* imply that the data will be shown. To show the data use
// ShowRawInstructions
func RawInstructions(instructions instructions.Instructions) RenderInstructionsTextOption {
	return func(o *renderInstructionsTextOptions) {
		o.instructions = instructions
	}
}

// Validate the options and panic if something is wrong
func (o renderInstructionsTextOptions) validate() {
	if o.showRawInstruction && o.instructions == nil {
		panic("RawInstructions(instructions) must be passed if ShowRawInstructions is used")
	}
}

// Render a textual representation of a program from a given reader.
// The reader must be correctly positioned over the instruction count
// of a sequence of instructions.
//
// See: RenderInstructionsText
func RenderInstructionsTextFromReader(reader io.Reader, opts ...RenderInstructionsTextOption) (string, error) {
	instructionList, err := instructions.DecodeInstructions(reader)
	if err != nil {
		return "", err
	}
	disassembled := instructions.Disassemble(instructionList)

	opts = append(opts, RawInstructions(instructionList))

	return RenderInstructionsText(disassembled, opts...), nil
}

// Render a textual representation of a program given a sequence
// of disassembled instructions
//
// Various options can be passed in order to show more information
// than would be possible with the default output which is
// compatible with Human Resource Machine (i.e. it can be pasted
// into the game)
func RenderInstructionsText(disassembled instructions.Disassembled, opts ...RenderInstructionsTextOption) string {
	var builder strings.Builder
	var options renderInstructionsTextOptions
	for _, opt := range opts {
		opt(&options)
	}
	options.validate()

	instNumPadding := int(math.Log10(float64(len(disassembled)))) + 1

	for i, diss := range disassembled {
		if options.showInstructionNumber {
			// print instruction number
			fmt.Fprintf(&builder, "%*d ", instNumPadding, i)
		}
		if options.showLineNumber {
			// print "line" number
			switch diss := diss.(type) {
			case instructions.DisassembleJumpTarget:
				fmt.Fprintf(&builder, "%*s ", instNumPadding, "")
			case instructions.DisassembleComment:
				fmt.Fprintf(&builder, "%*s ", instNumPadding, "")
			default:
				line := reflect.ValueOf(diss).FieldByName("Line").Int()
				fmt.Fprintf(&builder, "%*d ", instNumPadding, line)
			}
		}
		if options.showRawInstruction {
			inst := options.instructions[i]
			fmt.Fprintf(&builder, "%08X %08X %08X %08X ", inst.Comment, inst.Op, inst.Mode, inst.Arg)
		}
		// print label or opcode
		switch diss := diss.(type) {
		case instructions.DisassembleComment:
			fmt.Fprintf(&builder, "COMMENT %d", diss.Index)
		case instructions.DisassembleJumpTarget:
			fmt.Fprintf(&builder, "%s:", diss.Label)
		case instructions.DisassembleJumpInstruction:
			fmt.Fprintf(&builder, "%s %s", diss.Op.String(), diss.TargetLabel)
		case instructions.DisassembleArgInstruction:
			openBracket, closeBracket := "", ""
			if diss.Indirect {
				openBracket, closeBracket = "[", "]"
			}
			fmt.Fprintf(&builder, "%s %s%d%s", diss.Op.String(), openBracket, diss.Arg, closeBracket)
		case instructions.DisassembleInstruction:
			fmt.Fprint(&builder, diss.Op.String())
		}
		fmt.Fprintf(&builder, "\n")
	}

	return builder.String()
}

// Render a textual representation of a program's comments
// from a given reader. The reader must be correctly positioned
// over the comment count of a sequence of comments.
//
// See: RenderCommentsText
func RenderCommentsTextFromReader(reader io.ReadSeeker) (string, error) {
	rawComments, err := instructions.DecodeRawComments(reader)
	if err != nil {
		return "", err
	}

	return RenderCommentsText(rawComments), nil
}

// Render a sequence of raw comments as text. The rendered comments
// is compatible with the Human Resource Machine game (i.e. they
// can be pasted into the game) and should be appended to the rendered
// instructions. The returned text can be wrapped arbitrarily as long
// as the "DEFINE COMMENT xxx" text is not wrapped.
func RenderCommentsText(rawComments instructions.RawComments) string {
	var builder strings.Builder

	for commentIdx, comment := range rawComments {
		var b bytes.Buffer
		w, _ := zlib.NewWriterLevel(&b, 6)
		var i int
		var data [4]byte
		var dataBuffer bytes.Buffer
		binary.Write(&dataBuffer, binary.LittleEndian, uint32(len(comment)))
		w.Write(dataBuffer.Bytes())
		for i, data = range comment {
			w.Write(data[:])
		}
		for j := i; j < 1024/4-1; j++ {
			w.Write([]byte{0, 0, 0, 0})
		}
		w.Close()

		fmt.Fprintf(&builder, "DEFINE COMMENT %d\n", commentIdx)
		encodedComment := base64.StdEncoding.EncodeToString(b.Bytes())
		builder.WriteString(strings.TrimRight(encodedComment, "="))
		builder.WriteString(";\n\n")
	}

	return builder.String()
}
