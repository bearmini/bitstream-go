package bitstream

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

// Writer is a bit stream writer.
// It does not have io.Writer interface
type Writer struct {
	dst          io.Writer
	currByte     []uint8
	currBitIndex uint8 // MSB: 7, LSB: 0
}

// NewWriter creates a new Writer instance.
func NewWriter(dst io.Writer) *Writer {
	return &Writer{
		dst:          dst,
		currByte:     []byte{0},
		currBitIndex: 7,
	}
}

func (w *Writer) dump() string {
	return fmt.Sprintf("currByte: %02x, currBitIndex: %d", w.currByte[0], w.currBitIndex)
}

// WriteBit writes a single bit to the bit stream.
// Uses the LSB bit in `bit`.
func (w *Writer) WriteBit(bit uint8) error {
	if bit&0x01 != 0 {
		w.currByte[0] |= ((bit & 0x01) << w.currBitIndex)
	}

	if w.currBitIndex > 0 {
		w.currBitIndex--
		return nil
	}

	return w.Flush()
}

// WriteNBitsOfUint8 writes `nBits` bits to the bit stream.
// Uses n bits from `val`'s LSB
// i.e.)
//   if you have the following status of bit stream before calling WriteNBitsOfUint8,
//   currByte: 0101xxxxb
//   currBitIndex: 3
//
//   and if you calls WriteNBitsOfUint8(3, 0xaa),
//     where nBits == 3, val == 0xaa (10101010b)
//
//   WriteNBitsOfUint8 uses the 3 bits from `val`'s LSB, i.e.) xxxxx010b and as a result, status of the bit stream become:
//   currByte: 0101010xb (0101xxxxb | xxxx010xb)
//   currBitIndex: 0
func (w *Writer) WriteNBitsOfUint8(nBits, val uint8) error {
	// wb: bits can be written in currByte
	wb := w.currBitIndex + 1

	if nBits <= wb { // all the bits can be written in the currByte
		mask := uint8(1<<(nBits) - 1) // create a mask to make sure val has exactly n bits (to set 0's to upper bits)
		w.currByte[0] |= (val & mask) << (wb - nBits)
		if nBits == wb {
			return w.Flush()
		}
		w.currBitIndex -= nBits
		return nil
	}

	// need to separate val into 2 parts
	b1 := val >> (nBits - wb)       // part 1: can be written in the currByte
	b2 := val << (8 - (nBits - wb)) // part 2: should be written in the next byte (MSB aligned)
	b1Mask := uint8((1 << (w.currBitIndex + 1)) - 1)
	w.currByte[0] |= (b1 & b1Mask)
	w.Flush()
	w.currByte[0] = b2
	w.currBitIndex = 7 - (nBits - wb)

	return nil
}

// WriteUint8 writes a uint8 value to the stream.
func (w *Writer) WriteUint8(val uint8) error {
	return w.WriteNBitsOfUint8(8, val)
}

// Flush ensures the bufferred bits (bits not writen to the stream because it has less than 8 bits) to the destination writer.
func (w *Writer) Flush() error {
	nWritten, err := w.dst.Write(w.currByte)
	if err != nil {
		return err
	}
	if nWritten != 1 {
		return errors.New("unable to write 1 byte")
	}

	w.currByte[0] = 0x00
	w.currBitIndex = 7

	return nil
}
