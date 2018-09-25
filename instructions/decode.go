// Package instructions provides functions for decoding and disassembling
// Human Resource machine instructions stored in a profile
package instructions

import (
	"bytes"
	"encoding/binary"
	"io"
)

// A map of instruction indices to label names
type Labels map[uint32]string

// A decoded binary instruction
type Instruction struct {
	Comment uint32
	Op      uint32
	Mode    uint32 // or flag?
	Arg     uint32
}

// A list of binary instructions
type Instructions []Instruction

const (
	OP_INBOX      = 0x0
	OP_OUTBOX     = 0x1
	OP_COPY_FROM  = 0x2
	OP_COPY_TO    = 0x3
	OP_ADD        = 0x4
	OP_SUB        = 0x5
	OP_BUMP_MINUS = 0x6
	OP_BUMP_PLUS  = 0x7
	OP_JUMP       = 0x8
	OP_JUMP_ZERO  = 0x9
	OP_JUMP_NEG   = 0xA
	OP_JUMP_TGT   = 0xD
	MODE_DIRECT   = 0x1
	MODE_INDIRECT = 0x2
)

// Decode and return a sequence of instructions read from the
// passed in reader. The reader must be correctly positioned
// so that the first word read contains the instruction count
func DecodeInstructions(reader io.Reader) (Instructions, error) {
	var length uint32

	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		return nil, err
	}
	buffer := make([]byte, 4*4)
	instructions := make(Instructions, length)
	for i := uint32(0); i < length; i++ {
		if _, err := reader.Read(buffer); err != nil {
			return nil, err
		}
		b := bytes.NewBuffer(buffer)
		if err := binary.Read(b, binary.LittleEndian, &instructions[i]); err != nil {
			return nil, err
		}
	}

	return instructions, nil
}

// Given a label, return the next label. The starting label for Human
// Resource Machine programs should be "a"
func NextLabel(label string) string {
	carry := 1
	new_label := ""
	for i := len(label) - 1; i >= 0; i-- {
		digit := int(label[i]) + carry
		if digit > int('z') {
			digit = int('a')
			carry = 1
		} else {
			carry = 0
		}
		new_label = string(digit) + new_label
	}
	if carry == 1 {
		new_label = "a" + new_label
	}

	return new_label
}

// Given a list of instructions, return a map containing the symbolic
// label names for all jump targets
func MakeLabels(instructions Instructions) Labels {
	labels := make(Labels)

	label := "a"
	for i, inst := range instructions {
		if inst.Op == OP_JUMP_TGT {
			labels[uint32(i)] = label
			label = NextLabel(label)
		}
	}

	return labels
}

// A raw comment
type RawComment [][4]byte

// A sequence of raw comments
type RawComments []RawComment

// Decode binary comments found in reader into "raw" comments. RawComments
// are useful when rendering the comments back to a textual Human Resource
// Machine program representation. The reader must be correctly positioned
// so that the first word read contains the comment count
func DecodeRawComments(reader io.ReadSeeker) (RawComments, error) {
	var commentsLength uint32

	if err := binary.Read(reader, binary.LittleEndian, &commentsLength); err != nil {
		return nil, err
	}
	comments := make(RawComments, commentsLength)
	for commentIdx := uint32(0); commentIdx < commentsLength; commentIdx++ {
		var commentLength uint32

		if err := binary.Read(reader, binary.LittleEndian, &commentLength); err != nil {
			return nil, err
		}
		comments[commentIdx] = make(RawComment, commentLength)
		var i uint32
		for i = 0; i < commentLength; i++ {
			if err := binary.Read(reader, binary.LittleEndian, &comments[commentIdx][i]); err != nil {
				return nil, err
			}
		}
		skip := int64(1024 - commentLength*4)
		reader.Seek(skip, io.SeekCurrent)
	}
	return comments, nil
}

// A point in a comment, represented by an X and Y coordinate
// decoded from the RawComment
type CommentPoint struct {
	X uint16
	Y uint16
}

// A single unbroken graphical line, however a CommentLine of
// length one represents a dot
type CommentLine []CommentPoint

// A comment made up of one or more comment lines
type Comment []CommentLine

// A sequence of comments
type Comments []Comment

// Decode a sequence of RawComments into Comments. Comments
// are useful when, for example, rendering the comments
func DecodeComments(rawComments RawComments) (Comments, error) {
	comments := make(Comments, len(rawComments))
	for i, rawComment := range rawComments {
		comment := make(Comment, 0, len(rawComment)/2)
		line := make(CommentLine, 0, 5)
		for _, rawPoint := range rawComment {
			if rawPoint == [4]byte{0, 0, 0, 0} {
				comment = append(comment, line)
				line = make(CommentLine, 0, 5)
				continue
			}
			var point CommentPoint
			b := bytes.NewBuffer(rawPoint[:])
			if err := binary.Read(b, binary.LittleEndian, &point); err != nil {
				return nil, err
			}
			line = append(line, point)
		}
		comments[i] = comment
	}

	return comments, nil
}
