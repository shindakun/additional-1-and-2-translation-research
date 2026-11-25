// Package mips encodes the small set of MIPS II instructions the patches use.
// Register numbers follow the o32 names.
package mips

// Registers.
const (
	Zero = 0
	AT   = 1
	V0   = 2
	A0   = 4
	A1   = 5
	A2   = 6
	A3   = 7
	T0   = 8
	T1   = 9
	S0   = 16
	T9   = 25
	SP   = 29
	RA   = 31
)

func itype(op, rs, rt uint32, imm int32) uint32 {
	return op<<26 | rs<<21 | rt<<16 | uint32(uint16(imm))
}

func rtype(rs, rt, rd, fn uint32) uint32 { return rs<<21 | rt<<16 | rd<<11 | fn }

// Nop encodes nop.
func Nop() uint32 { return 0 }

// Addiu encodes addiu rt, rs, imm.
func Addiu(rt, rs uint32, imm int32) uint32 { return itype(9, rs, rt, imm) }

// Ori encodes ori rt, rs, imm.
func Ori(rt, rs uint32, imm uint16) uint32 { return itype(0xD, rs, rt, int32(imm)) }

// Lui encodes lui rt, imm.
func Lui(rt uint32, imm uint16) uint32 { return itype(0xF, 0, rt, int32(imm)) }

// Lw encodes lw rt, off(base).
func Lw(rt, base uint32, off int32) uint32 { return itype(0x23, base, rt, off) }

// Sw encodes sw rt, off(base).
func Sw(rt, base uint32, off int32) uint32 { return itype(0x2B, base, rt, off) }

// Addu encodes addu rd, rs, rt.
func Addu(rd, rs, rt uint32) uint32 { return rtype(rs, rt, rd, 0x21) }

// Or encodes or rd, rs, rt.
func Or(rd, rs, rt uint32) uint32 { return rtype(rs, rt, rd, 0x25) }

// Jr encodes jr rs.
func Jr(rs uint32) uint32 { return rtype(rs, 0, 0, 0x08) }

// Jalr encodes jalr rs, linking to ra.
func Jalr(rs uint32) uint32 { return rtype(rs, 0, RA, 0x09) }

// Beq encodes beq rs, rt, off (branch to pc+4+off*4).
func Beq(rs, rt uint32, off int32) uint32 { return itype(4, rs, rt, off) }

// Mtc1 moves a general register to floating-point register fs.
func Mtc1(rt, fs uint32) uint32 { return 0x11<<26 | 4<<21 | rt<<16 | fs<<11 }

// Bal branches and links to pc+4+off*4.
func Bal(off int32) uint32 { return itype(1, 0, 0x11, off) }

// Jal encodes a call to an absolute address (the loader relocates it).
func Jal(target uint32) uint32 { return 3<<26 | (target>>2)&0x03FFFFFF }

// Program collects instructions and data at a base address.
type Program struct {
	Base  uint32
	Words []uint32
}

// Emit appends instructions and returns the address of the first.
func (p *Program) Emit(w ...uint32) uint32 {
	at := p.Base + uint32(len(p.Words))*4
	p.Words = append(p.Words, w...)
	return at
}

// PC returns the address of the next instruction.
func (p *Program) PC() uint32 { return p.Base + uint32(len(p.Words))*4 }

// Bytes returns the program as little-endian bytes.
func (p *Program) Bytes() []byte {
	b := make([]byte, len(p.Words)*4)
	for i, w := range p.Words {
		b[i*4], b[i*4+1], b[i*4+2], b[i*4+3] = byte(w), byte(w>>8), byte(w>>16), byte(w>>24)
	}
	return b
}

// LoadConst emits lui/ori to put a 32-bit constant in rt.
func (p *Program) LoadConst(rt, v uint32) {
	p.Emit(Lui(rt, uint16(v>>16)), Ori(rt, rt, uint16(v)))
}
