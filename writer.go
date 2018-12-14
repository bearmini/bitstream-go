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
// `nBits` must be less than or equal to 8, otherwise returns an error.
//
// This function uses n bits from `val`'s LSB.
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
	if nBits == 0 {
		return nil
	}

	if nBits > 8 {
		return errors.New("nBits too large for uint8")
	}

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
	err := w.Flush()
	if err != nil {
		return err
	}
	w.currByte[0] = b2
	w.currBitIndex = 7 - (nBits - wb)

	return nil
}

// WriteUint8 writes a uint8 value to the bit stream.
func (w *Writer) WriteUint8(val uint8) error {
	return w.WriteNBitsOfUint8(8, val)
}

// WriteNBitsOfUint16 writes `nBits` bits to the bit stream.
// `nBits` must be less than or equal to 16, otherwise returns an error.
func (w *Writer) WriteNBitsOfUint16(nBits uint8, val uint16) error {
	if nBits == 0 {
		return nil
	}

	if nBits <= 8 {
		return w.WriteNBitsOfUint8(nBits, uint8(val))
	}

	if nBits > 16 {
		return errors.New("nBits too large for uint16")
	}

	// wb: bits can be written in currByte
	wb := w.currBitIndex + 1

	// 16 bits may be distributed in 3 bytes
	b1Bits := wb
	b2Bits := uint8(nBits - b1Bits)
	b3Bits := uint8(0)
	if b2Bits > 8 {
		b3Bits = b2Bits - 8
		b2Bits = 8
	}

	b1Mask := uint16(((1 << b1Bits) - 1) << (b2Bits + b3Bits))
	b2Mask := uint16(((1 << b2Bits) - 1) << b3Bits)
	b3Mask := uint16((1 << b3Bits) - 1)

	b1 := uint8((val & b1Mask) >> (b2Bits + b3Bits))
	b2 := uint8(((val & b2Mask) >> b3Bits) << (8 - b2Bits)) // left aligned
	b3 := uint8((val & b3Mask) << (8 - b3Bits))             // left aligned

	w.currByte[0] |= b1
	err := w.Flush()
	if err != nil {
		return err
	}

	if b3Bits == 0 {
		w.currByte[0] = b2
		if b2Bits == 8 {
			return w.Flush()
		}
		w.currBitIndex = 7 - b2Bits
		return nil
	}

	w.currByte[0] = b2
	err = w.Flush()
	if err != nil {
		return err
	}
	w.currByte[0] = b3
	w.currBitIndex = 7 - b3Bits

	return nil
}

// WriteUint16 writes a uint16 value to the bit stream.
func (w *Writer) WriteUint16(val uint16) error {
	return w.WriteNBitsOfUint16(16, val)
}

// WriteNBitsOfUint32 writes `nBits` bits to the bit stream.
// `nBits` must be less than or equal to 32, otherwise returns an error.
func (w *Writer) WriteNBitsOfUint32(nBits uint8, val uint32) error {
	if nBits == 0 {
		return nil
	}

	if nBits <= 16 {
		return w.WriteNBitsOfUint16(nBits, uint16(val))
	}

	if nBits > 32 {
		return errors.New("nBits too large for uint32")
	}

	// wb: bits can be written in currByte
	wb := w.currBitIndex + 1

	// 32 bits may be distributed in 5 bytes
	b1Bits := wb
	b2Bits := uint8(8)
	b3Bits := uint8(nBits - 8 - wb)
	b4Bits := uint8(0)
	b5Bits := uint8(0)
	if b3Bits > 8 {
		b4Bits = b3Bits - 8
		if b4Bits > 8 {
			b5Bits = b4Bits - 8
			b4Bits = 8
		}
		b3Bits = 8
	}

	b1Mask := uint32(((1 << b1Bits) - 1) << (b2Bits + b3Bits + b4Bits + b5Bits))
	b2Mask := uint32(((1 << b2Bits) - 1) << (b3Bits + b4Bits + b5Bits))
	b3Mask := uint32(((1 << b3Bits) - 1) << (b4Bits + b5Bits))
	b4Mask := uint32(((1 << b4Bits) - 1) << b5Bits)
	b5Mask := uint32((1 << b5Bits) - 1)

	b1 := uint8((val & b1Mask) >> (b2Bits + b3Bits + b4Bits + b5Bits))
	b2 := uint8(((val & b2Mask) >> (b3Bits + b4Bits + b5Bits)) << (8 - b2Bits)) // left aligned
	b3 := uint8(((val & b3Mask) >> (b4Bits + b5Bits)) << (8 - b3Bits))          // left aligned
	b4 := uint8(((val & b4Mask) >> b5Bits) << (8 - b4Bits))                     // left aligned
	b5 := uint8((val & b5Mask) << (8 - b5Bits))                                 // left aligned

	w.currByte[0] |= b1
	err := w.Flush()
	if err != nil {
		return err
	}

	w.currByte[0] = b2
	err = w.Flush()
	if err != nil {
		return err
	}

	w.currByte[0] = b3
	if b3Bits == 8 {
		err = w.Flush()
		if err != nil {
			return err
		}
	}
	if b4Bits == 0 {
		if b3Bits != 8 {
			w.currBitIndex = 7 - b3Bits
		}
		return nil
	}

	w.currByte[0] = b4
	if b4Bits == 8 {
		err = w.Flush()
		if err != nil {
			return err
		}
	}
	if b5Bits == 0 {
		if b4Bits != 8 {
			w.currBitIndex = 7 - b4Bits
		}
		return nil
	}

	w.currByte[0] = b5
	w.currBitIndex = 7 - b5Bits

	return nil
}

// WriteUint32 writes a uint32 value to the bit stream.
func (w *Writer) WriteUint32(val uint32) error {
	return w.WriteNBitsOfUint32(32, val)
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

func hex(x uint32) string {
	return fmt.Sprintf("%#08x", x)
}
