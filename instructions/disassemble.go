package instructions

// A type representing an opcode to instruction mnemonic map
type instrunctionMnemonics map[OpCode]string

// A map of mnemonics
var InstrunctionMnemonics = instrunctionMnemonics{
	OP_INBOX:      "INBOX",
	OP_OUTBOX:     "OUTBOX",
	OP_COPY_FROM:  "COPYFROM",
	OP_COPY_TO:    "COPYTO",
	OP_ADD:        "ADD",
	OP_SUB:        "SUB",
	OP_BUMP_MINUS: "BUMPDN",
	OP_BUMP_PLUS:  "BUMPUP",
	OP_JUMP:       "JUMP",
	OP_JUMP_ZERO:  "JUMPZ",
	OP_JUMP_NEG:   "JUMPN",
}

// Membership test for InstrunctionMnemonics
func (im instrunctionMnemonics) Member(op OpCode) bool {
	_, ok := im[op]
	return ok
}

// A type representing opcodes that take an argument
type instructionsWithArg map[OpCode]struct{}

// A map containing opcodes that take an argument
var InstructionsWithArg = instructionsWithArg{
	OP_COPY_FROM:  {},
	OP_COPY_TO:    {},
	OP_ADD:        {},
	OP_SUB:        {},
	OP_BUMP_MINUS: {},
	OP_BUMP_PLUS:  {},
}

// Membership test for InstructionsWithArg
func (im instructionsWithArg) Member(op OpCode) bool {
	_, ok := im[op]
	return ok
}

// A type representing opcodes that have a target label
type instructionsWithLabel map[OpCode]struct{}

// A map containing opcodes that have a target label
var InstructionsWithLabel = instructionsWithLabel{
	OP_JUMP:      {},
	OP_JUMP_ZERO: {},
	OP_JUMP_NEG:  {},
}

// Membership test for InstructionsWithLabel
func (im instructionsWithLabel) Member(op OpCode) bool {
	_, ok := im[op]
	return ok
}

// An interface for different kinds of disassembled
// instructions.
type DisassembleInterface interface {
	isDissasemble()
}

// A comment (implements DisassembleInterface)
type DisassembleComment struct {
	Index uint32
}

func (d DisassembleComment) isDissasemble() {}

// A jump target (implements DisassembleInterface)
type DisassembleJumpTarget struct {
	Label  string
	Jumpee int
}

func (d DisassembleJumpTarget) isDissasemble() {}

// Binary opcode
type OpCode uint32

// Return the mnemonic for the opcode
func (o OpCode) String() string {
	return InstrunctionMnemonics[o]
}

// An instruction taking no arguments (implements DisassembleInterface)
type DisassembleInstruction struct {
	Line int
	Op   OpCode
}

func (d DisassembleInstruction) isDissasemble() {}

// An jump instruction (implements DisassembleInterface)
type DisassembleJumpInstruction struct {
	DisassembleInstruction
	TargetLabel string
	Target      int
}

func (d DisassembleJumpInstruction) isDissasemble() {}

// An instruction taking one argument (implements DisassembleInterface)
type DisassembleArgInstruction struct {
	DisassembleInstruction
	Arg      uint32
	Indirect bool
}

func (d DisassembleArgInstruction) isDissasemble() {}

// A list of disassembled instructions
type Disassembled []DisassembleInterface

// Given a sequence of instructions, return the disassembled
// instructions
func Disassemble(instructions Instructions) Disassembled {
	labels := MakeLabels(instructions)
	disassembled := make(Disassembled, len(instructions))
	instNum := 1
	for i, inst := range instructions {
		opCode := OpCode(inst.Op)
		switch {
		case inst.Comment > 0:
			disassembled[i] = DisassembleComment{inst.Op}
			continue // Does not increment instNum
		case opCode == OP_JUMP_TGT:
			continue // Set by JUMP instruction; Does not increment instNum
		case InstructionsWithLabel.Member(opCode):
			label := labels[inst.Arg]
			disassembled[i] = DisassembleJumpInstruction{
				DisassembleInstruction{instNum, opCode}, label, int(inst.Arg)}
			disassembled[inst.Arg] = DisassembleJumpTarget{label, i}
		case InstructionsWithArg.Member(opCode):
			disassembled[i] = DisassembleArgInstruction{
				DisassembleInstruction{instNum, opCode}, inst.Arg, inst.Mode == MODE_INDIRECT}
		case InstrunctionMnemonics.Member(opCode):
			disassembled[i] = DisassembleInstruction{instNum, opCode}
		}

		instNum++
	}

	return disassembled
}
