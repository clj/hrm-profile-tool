package render

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	svg "github.com/ajstarks/svgo"
	"github.com/clj/hrm-profile-tool/instructions"
)

type Colour string

var (
	ioColour      = Colour("rgb(156, 182, 92)")
	jumpColour    = Colour("rgb(141, 141, 193)")
	copyColour    = Colour("rgb(200, 106, 84)")
	arithColour   = Colour("rgb(197, 139, 97)")
	commentColour = Colour("rgb(227, 219, 198)")
	canvasColour  = Colour("rgb(188, 160, 139)")
	textColour    = Colour("rgb(68, 80, 37)")
	lineNoColour  = Colour("rgb(125, 106, 92)")
)

func (c Colour) fill() string {
	return fmt.Sprintf("fill:%s", c)
}

type TextStyle string

var instTextStyle = TextStyle("font-family:'Arial Black';font-size:%s;" + textColour.fill())
var lineNoTextStyle = TextStyle("font-family:'Arial Black';font-size:%s;" + lineNoColour.fill())

func (t TextStyle) Render(fontSize string) string {
	return fmt.Sprintf(string(t), fontSize)
}

type SVGMnemonics struct {
	Width    int
	Mnemonic string
	Colour   Colour
}

var svgInstrunctionMnemonics = map[instructions.OpCode]SVGMnemonics{
	instructions.OP_INBOX:      {90, "inbox", ioColour},
	instructions.OP_OUTBOX:     {90, "outbox", ioColour},
	instructions.OP_COPY_FROM:  {110, "copyfrom", copyColour},
	instructions.OP_COPY_TO:    {90, "copyto", copyColour},
	instructions.OP_ADD:        {60, "add", arithColour},
	instructions.OP_SUB:        {60, "sub", arithColour},
	instructions.OP_BUMP_MINUS: {85, "bump -", arithColour},
	instructions.OP_BUMP_PLUS:  {85, "bump +", arithColour},
	instructions.OP_JUMP:       {75, "jump", jumpColour},
	instructions.OP_JUMP_ZERO:  {95, "jump", jumpColour},
	instructions.OP_JUMP_NEG:   {120, "jump", jumpColour},
}

var svgJumpConditions = map[instructions.OpCode]string{
	instructions.OP_JUMP:      "",
	instructions.OP_JUMP_ZERO: "zero",
	instructions.OP_JUMP_NEG:  "negative",
}

func absInt(n int) int {
	y := n >> strconv.IntSize
	return (n ^ y) - y
}

func instruction(canvas *svg.SVG, x, y, w, h int, style, op string) {
	canvas.Gtransform(fmt.Sprintf("translate(%d, %d)", x, y))
	canvas.Roundrect(0, 0, w, h, 2, 2, style, `filter="url(#dropShadow)"`)
	if op != "" {
		fmt.Fprintf(canvas.Writer, `<svg width="%d" height="%d">`+"\n", w, h)
		canvas.Text(
			w/2, h/2, op, instTextStyle.Render("16px"),
			`alignment-baseline="central" text-anchor="middle"`)
		canvas.End()
	}
	canvas.Gend()
}

func jumpInstruction(canvas *svg.SVG, x, y, w, h int, style, op, condition string) {
	canvas.Gtransform(fmt.Sprintf("translate(%d, %d)", x, y))
	canvas.Roundrect(0, 0, w, h, 2, 2, style, `filter="url(#dropShadow)"`)
	fmt.Fprintf(canvas.Writer, `<svg width="%d" height="%d">`+"\n", w, h)
	if condition != "" {
		canvas.Text(
			15, h/2, op,
			instTextStyle.Render("16px"), `alignment-baseline="central" text-anchor="left"`)
		canvas.Text(
			15+45, h/3, "if",
			instTextStyle.Render("10px"), `alignment-baseline="central" text-anchor="left"`)
		canvas.Text(
			15+45, (h/3)*2, condition,
			instTextStyle.Render("10px"), `alignment-baseline="central" text-anchor="left"`)
	} else {
		canvas.Text(
			w/2, h/2, op,
			instTextStyle.Render("16px"), `alignment-baseline="central" text-anchor="middle"`)
	}
	canvas.End()
	canvas.Gend()
}

func argument(canvas *svg.SVG, x, y, w, h int, style string, arg uint32, indirect bool) {
	canvas.Gtransform(fmt.Sprintf("translate(%d, %d)", x, y))
	canvas.Roundrect(0, 0, w, h, 2, 2, style, `filter="url(#dropShadow)"`)
	fmt.Fprintf(canvas.Writer, `<svg width="%d" height="%d">`+"\n", w, h)
	// XXX: Deal with defined label
	var strArg string
	if indirect {
		strArg = fmt.Sprintf("[%d]", arg)
	} else {
		strArg = fmt.Sprintf("%d", arg)
	}
	canvas.Text(
		w/2, h/2, strArg,
		instTextStyle.Render("22px"), `alignment-baseline="central" text-anchor="middle"`)
	canvas.End()
	canvas.Gend()
}

func lineNumber(canvas *svg.SVG, x, y, width, height, lineNumber int) {
	canvas.Text(
		(x+width)/2, y+height/2, fmt.Sprintf("%02d", lineNumber),
		lineNoTextStyle.Render("16px"), `alignment-baseline="central" text-anchor="middle"`)
}

func comment(canvas *svg.SVG, x, y, w, h int, comment instructions.Comment) {
	style := commentColour.fill()
	canvas.Gtransform(fmt.Sprintf("translate(%d, %d)", x, y))
	canvas.Roundrect(0, 0, w, h, 2, 2, style, `filter="url(#dropShadow)"`)
	fmt.Fprintf(canvas.Writer, `<svg width="%d" height="%d">`+"\n", w, h)
	canvas.Def()
	canvas.ClipPath(`id="clipping-rect"`)
	canvas.Roundrect(0, 0, w, h, 2, 2)
	canvas.ClipEnd()
	canvas.DefEnd()
	scaleX := (float64(w) / math.MaxUint16)
	scaleY := (float64(h) / math.MaxUint16)
	for _, line := range comment {
		if len(line) == 1 {
			point := line[0]
			canvas.Circle(int(float64(point.X)*scaleX), int(float64(point.Y)*scaleY), 2, `clip-path="url(#clipping-rect)"`)
		} else {
			xs := make([]int, len(line))
			ys := make([]int, len(line))
			for i, point := range line {
				xs[i], ys[i] = int(float64(point.X)*scaleX), int(float64(point.Y)*scaleY)
			}
			canvas.Polyline(xs, ys, `fill="none" stroke="black" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" clip-path="url(#clipping-rect)"`)
		}
	}
	canvas.End()
	canvas.Gend()
}

// Render an SVG representation of a program. The reader must be correctly
// positioned instruction count of program.
//
// See: RenderSVG
func RenderSVGFromReader(reader io.ReadSeeker) (string, error) {
	start, _ := reader.Seek(0, io.SeekCurrent)
	instructionList, err := instructions.DecodeInstructions(reader)
	if err != nil {
		return "", err
	}
	disassembled := instructions.Disassemble(instructionList)

	reader.Seek(start+4100, io.SeekStart)
	rawComments, err := instructions.DecodeRawComments(reader)
	if err != nil {
		return "", err
	}
	comments, err := instructions.DecodeComments(rawComments)
	if err != nil {
		return "", err
	}

	return RenderSVG(disassembled, comments), nil
}

// Render a sequence of disassembled instructions and comments into an SVG. The rendered
// SVG emulates the style of the game's display of instructions.
func RenderSVG(disassembled instructions.Disassembled, comments instructions.Comments) string {
	var builder strings.Builder

	canvas := svg.New(&builder)

	lineNumberColumnWidth := 35
	instXOffset, instYOffset, instYStep, instHeight := 10, 10, 30, 25
	commentYStep, commentHeight, commentWidth := 45, 40, 120
	canvasWidth := 300
	canvasHeight := len(disassembled)*instYStep + instYOffset*2 + len(comments)*(commentYStep-instYStep)
	targetLabelWidth := 75
	canvas.Start(canvasWidth, canvasHeight)

	canvas.Def()
	canvas.Filter("dropShadow", `width="200%" height="200%"`)
	canvas.FeOffset(svg.Filterspec{In: "SourceAlpha", Result: "offOut"}, 1, 1)
	canvas.FeColorMatrix(
		svg.Filterspec{In: "offOut", Result: "matrixOut"},
		[...]float64{0.2, 0, 0, 0, 0, 0, 0.2, 0, 0, 0, 0, 0, 0.2, 0, 0, 0, 0, 0, 0.5, 0}, `mode="normal"`)
	canvas.FeGaussianBlur(svg.Filterspec{In: "matrixOut", Result: "blurOut"}, 1, 1)
	canvas.FeBlend(svg.Filterspec{In: "SourceGraphic", In2: "blurOut"}, `mode="normal"`)
	canvas.Fend()
	canvas.Marker("arrow", 3, 3, 10, 10)
	canvas.Path("M10 0 10 6 1 3z", jumpColour.fill())
	canvas.MarkerEnd()
	canvas.LinearGradient("lineNumberColumn", 0, 0, 100, 0, []svg.Offcolor{
		{0, "rgb(140,119,104)", 1.0},
		{40, "rgb(172,146,127)", 1.0},
		{100, "rgb(172,146,127)", 1.0}})
	canvas.DefEnd()

	canvas.Rect(0, 0, canvasWidth, canvasHeight, canvasColour.fill())
	canvas.Rect(0, 0, lineNumberColumnWidth, canvasHeight, "fill:url(#lineNumberColumn)")

	// calculate comments up to the i'th instruction
	numComments := 0
	commentCount := make([]int, len(disassembled))
	for i, diss := range disassembled {
		commentCount[i] = numComments
		switch diss.(type) {
		case instructions.DisassembleComment:
			numComments++
		}
	}

	// draw jump lines first, which should be under instructions
	for i, diss := range disassembled {
		switch diss := diss.(type) {
		case instructions.DisassembleJumpInstruction:
			mnemonic := svgInstrunctionMnemonics[diss.Op]
			targetCommentOffset := commentCount[diss.Target] * (commentYStep - instYStep)
			currentCommentOffset := commentCount[i] * (commentYStep - instYStep)
			sx := lineNumberColumnWidth + instXOffset + mnemonic.Width
			sy := instYOffset + i*instYStep + currentCommentOffset + instHeight/2
			cx := canvasWidth
			cy := sy
			ex := lineNumberColumnWidth + instXOffset + targetLabelWidth + 10
			ey := instYOffset + diss.Target*instYStep + targetCommentOffset + instHeight/2
			px := canvasWidth
			py := ey
			canvas.Bezier(
				sx, sy, cx, cy, px, py, ex, ey,
				`fill="none" stroke="rgb(141, 141, 193)" stroke-width="3"`+
					` marker-end="url(#arrow)" filter="url(#dropShadow)"`)
		}
	}
	// draw instructions
	for i, diss := range disassembled {
		instX := lineNumberColumnWidth + instXOffset
		instY := instYOffset + i*instYStep + commentCount[i]*(commentYStep-instYStep)
		switch diss := diss.(type) {
		case instructions.DisassembleComment:
			comment(canvas, instX, instY, commentWidth, commentHeight, comments[diss.Index])
		case instructions.DisassembleJumpTarget:
			instruction(canvas, instX, instY, targetLabelWidth, instHeight, jumpColour.fill(), "")
		case instructions.DisassembleJumpInstruction:
			lineNumber(canvas, 0, instY, lineNumberColumnWidth, instHeight, diss.Line)
			mnemonic := svgInstrunctionMnemonics[diss.Op]
			condition := svgJumpConditions[diss.Op]
			jumpInstruction(
				canvas, instX, instY, mnemonic.Width, instHeight,
				mnemonic.Colour.fill(), mnemonic.Mnemonic, condition)
		case instructions.DisassembleArgInstruction:
			lineNumber(canvas, 0, instY, lineNumberColumnWidth, instHeight, diss.Line)
			mnemonic := svgInstrunctionMnemonics[diss.Op]
			instruction(
				canvas, instX, instY, mnemonic.Width, instHeight,
				mnemonic.Colour.fill(), mnemonic.Mnemonic)
			argument(
				canvas, instX+mnemonic.Width+10, instY, 50, instHeight,
				mnemonic.Colour.fill(), diss.Arg, diss.Indirect)
		case instructions.DisassembleInstruction:
			lineNumber(canvas, 0, instY, lineNumberColumnWidth, instHeight, diss.Line)
			mnemonic := svgInstrunctionMnemonics[diss.Op]
			instruction(
				canvas, instX, instY, mnemonic.Width, instHeight,
				mnemonic.Colour.fill(), mnemonic.Mnemonic)
		}
	}
	canvas.End()

	return builder.String()
}
