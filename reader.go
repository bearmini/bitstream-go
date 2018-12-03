package bitstream

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

const (
	DefaultBufferSize = 1024
)

type Reader struct {
	src           io.Reader
	srcEOF        bool
	buf           []byte
	bufLen        int
	currByteIndex int   // starts from 0
	currBitIndex  uint8 // MSB: 7, LSB: 0
	opt           *Options
}

type Options struct {
	BufferSize uint
}

func (opt *Options) GetBufferSize() uint {
	if opt == nil || opt.BufferSize == 0 {
		return DefaultBufferSize
	}
	return opt.BufferSize
}

func NewReader(src io.Reader, opt *Options) *Reader {
	return &Reader{
		src:           src,
		srcEOF:        false,
		buf:           nil,
		bufLen:        0,
		currByteIndex: 0,
		currBitIndex:  7,
		opt:           opt,
	}
}

func (r *Reader) dump() {
	fmt.Printf("srcEOF=%t, bufLen=%d, currByteIndex=%d, currBitIndex=%d\n", r.srcEOF, r.bufLen, r.currByteIndex, r.currBitIndex)
}

func (r *Reader) isBufEmpty() bool {
	if r.buf == nil {
		return true
	}

	if r.currByteIndex >= r.bufLen {
		return true
	}

	return false
}

func (r *Reader) fillBuf() error {
	buf := make([]byte, r.opt.GetBufferSize())
	n, err := r.src.Read(buf[:])
	if err != nil {
		return err
	}

	r.buf = buf
	r.bufLen = n
	r.currByteIndex = 0
	r.currBitIndex = 7
	return nil
}

func (r *Reader) fillBufIfNeeded() error {
	if !r.isBufEmpty() {
		return nil
	}
	return r.fillBuf()
}

// ex. startBitIndex = 5, nBits = 3
// b7 b6 b5 b4 b3 b2 b1 b0
//  0  0  1  1  1  0  0  0
func (r *Reader) getBitMaskByte(startBitIndex, nBits uint8) (byte, error) {
	if (7-startBitIndex)+nBits > 8 {
		return 0, errors.New("unable to create mask")
	}
	mask := uint8(0)
	for i := uint8(0); i < nBits; i++ {
		b := uint8(1 << (startBitIndex - i))
		mask = mask | b
	}

	return mask, nil
}

func (r *Reader) forwardIndecies(nBits uint8) {
	if nBits <= r.currBitIndex {
		r.currBitIndex -= nBits
		return
	}

	nBits = nBits - r.currBitIndex
	nBytes := int(nBits/8) + 1
	r.currByteIndex += nBytes

	bitsToGo := (nBits % 8)
	r.currBitIndex = 8 - bitsToGo
}

func (r *Reader) ReadBit() (byte, error) {
	err := r.fillBufIfNeeded()
	if err != nil {
		return 0, err
	}

	b := r.buf[r.currByteIndex]
	bm, err := r.getBitMaskByte(r.currBitIndex, 1)
	if err != nil {
		return 0, err
	}

	result := (b & bm) >> r.currBitIndex
	r.forwardIndecies(1)
	return result, nil
}

func (r *Reader) mustReadNBitsInCurrentByte(nBits uint8) byte {
	if nBits == 0 {
		return 0
	}

	if r.currBitIndex < (nBits - 1) {
		panic(fmt.Sprintf("%+v", errors.New("insufficient bits to read")))
	}

	b := r.buf[r.currByteIndex]
	bm, err := r.getBitMaskByte(r.currBitIndex, nBits)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	result := (b & bm) >> (r.currBitIndex - (nBits - 1))
	r.forwardIndecies(nBits)
	return result
}

func (r *Reader) ReadNBitsAsUint8(nBits uint8) (uint8, error) {
	if nBits == 0 {
		return 0, nil
	}

	err := r.fillBufIfNeeded()
	if err != nil {
		return 0, err
	}

	// remaining bits in current byte
	rb := r.currBitIndex + 1
	if nBits <= rb {
		b := r.mustReadNBitsInCurrentByte(nBits)
		return b, nil
	}

	nBits1 := rb
	nBits2 := nBits - rb

	b1 := r.mustReadNBitsInCurrentByte(nBits1)
	b2, err := r.ReadNBitsAsUint8(nBits2)
	if err != nil {
		return 0, err
	}

	return (b1 << nBits2) | b2, nil
}

func (r *Reader) ReadNBitsAsUint16BE(nBits uint8) (uint16, error) {
	if nBits == 0 {
		return 0, nil
	}

	if nBits <= 8 {
		v, err := r.ReadNBitsAsUint8(nBits)
		return uint16(v), err
	}

	if nBits > 16 {
		return 0, errors.New("nBits too large for uint16")
	}

	err := r.fillBufIfNeeded()
	if err != nil {
		return 0, err
	}

	// remaining bits in current byte
	rb := r.currBitIndex + 1
	nBits1 := rb
	nBits2 := nBits - rb
	nBits3 := uint8(0)
	if nBits2 > 8 {
		nBits3 = nBits2 - 8
		nBits2 = 8
	}

	b1 := r.mustReadNBitsInCurrentByte(nBits1)
	b2, err := r.ReadNBitsAsUint8(nBits2)
	if err != nil {
		return 0, err
	}
	b3, err := r.ReadNBitsAsUint8(nBits3)
	if err != nil {
		return 0, err
	}

	return (uint16(b1) << (nBits2 + nBits3)) | (uint16(b2) << nBits3) | uint16(b3), nil
}

func (r *Reader) ReadNBitsAsUint32BE(nBits uint8) (uint32, error) {
	if nBits == 0 {
		return 0, nil
	}

	if nBits <= 16 {
		v, err := r.ReadNBitsAsUint16BE(nBits)
		return uint32(v), err
	}

	if nBits > 32 {
		return 0, errors.New("nBits too large for uint32")
	}

	err := r.fillBufIfNeeded()
	if err != nil {
		return 0, err
	}

	// remaining bits in current byte
	rb := r.currBitIndex + 1
	nBits1 := rb
	nBits2 := uint8(8)
	nBits3 := nBits - rb - 8
	nBits4 := uint8(0)
	nBits5 := uint8(0)
	if nBits3 > 8 {
		nBits4 = nBits3 - 8
		if nBits4 > 8 {
			nBits5 = nBits4 - 8
			nBits4 = 8
		}
		nBits3 = 8
	}

	b1 := r.mustReadNBitsInCurrentByte(nBits1)
	b2, err := r.ReadNBitsAsUint8(nBits2)
	if err != nil {
		return 0, err
	}
	b3, err := r.ReadNBitsAsUint8(nBits3)
	if err != nil {
		return 0, err
	}
	b4, err := r.ReadNBitsAsUint8(nBits4)
	if err != nil {
		return 0, err
	}
	b5, err := r.ReadNBitsAsUint8(nBits5)
	if err != nil {
		return 0, err
	}

	return (uint32(b1) << (nBits2 + nBits3 + nBits4 + nBits5)) | (uint32(b2) << (nBits3 + nBits4 + nBits5)) | (uint32(b3) << (nBits4 + nBits5)) | (uint32(b4) << (nBits5)) | uint32(b5), nil
}

func (r *Reader) ReadNBitsAsUint64BE(nBits uint8) (uint64, error) {
	if nBits == 0 {
		return 0, nil
	}

	if nBits <= 32 {
		v, err := r.ReadNBitsAsUint32BE(nBits)
		return uint64(v), err
	}

	if nBits > 64 {
		return 0, errors.New("nBits too large for uint64")
	}

	err := r.fillBufIfNeeded()
	if err != nil {
		return 0, err
	}

	// remaining bits in current byte
	rb := r.currBitIndex + 1
	nBits1 := rb
	nBits2 := uint8(8)
	nBits3 := uint8(8)
	nBits4 := uint8(8)
	nBits5 := nBits - rb - 24
	nBits6 := uint8(0)
	nBits7 := uint8(0)
	nBits8 := uint8(0)
	nBits9 := uint8(0)
	if nBits5 > 8 {
		nBits6 = nBits5 - 8
		if nBits6 > 8 {
			nBits7 = nBits6 - 8
			if nBits7 > 8 {
				nBits8 = nBits7 - 8
				if nBits8 > 8 {
					nBits9 = nBits8 - 8
					nBits8 = 8
				}
				nBits7 = 8
			}
			nBits6 = 8
		}
		nBits5 = 8
	}

	b1 := r.mustReadNBitsInCurrentByte(nBits1)
	b2, err := r.ReadNBitsAsUint8(nBits2)
	if err != nil {
		return 0, err
	}
	b3, err := r.ReadNBitsAsUint8(nBits3)
	if err != nil {
		return 0, err
	}
	b4, err := r.ReadNBitsAsUint8(nBits4)
	if err != nil {
		return 0, err
	}
	b5, err := r.ReadNBitsAsUint8(nBits5)
	if err != nil {
		return 0, err
	}
	b6, err := r.ReadNBitsAsUint8(nBits6)
	if err != nil {
		return 0, err
	}
	b7, err := r.ReadNBitsAsUint8(nBits7)
	if err != nil {
		return 0, err
	}
	b8, err := r.ReadNBitsAsUint8(nBits8)
	if err != nil {
		return 0, err
	}
	b9, err := r.ReadNBitsAsUint8(nBits9)
	if err != nil {
		return 0, err
	}

	return (uint64(b1) << (nBits2 + nBits3 + nBits4 + nBits5 + nBits6 + nBits7 + nBits8 + nBits9)) |
		(uint64(b2) << (nBits3 + nBits4 + nBits5 + nBits6 + nBits7 + nBits8 + nBits9)) |
		(uint64(b3) << (nBits4 + nBits5 + nBits6 + nBits7 + nBits8 + nBits9)) |
		(uint64(b4) << (nBits5 + nBits6 + nBits7 + nBits8 + nBits9)) |
		(uint64(b5) << (nBits6 + nBits7 + nBits8 + nBits9)) |
		(uint64(b6) << (nBits7 + nBits8 + nBits9)) |
		(uint64(b7) << (nBits8 + nBits9)) |
		(uint64(b8) << (nBits9)) |
		uint64(b9), nil
}

type ReadOptions struct {
	AlignRight bool
	PadOne     bool
}

func (r *Reader) ReadNBits(nBits uint8, opt *ReadOptions) ([]byte, error) {
	if nBits == 0 {
		return nil, nil
	}

	err := r.fillBufIfNeeded()
	if err != nil {
		return nil, err
	}

	padOne := (opt != nil && opt.PadOne)
	alignRight := (opt != nil && opt.AlignRight)

	maxByteLen := (nBits / 8) + 1
	result := make([]byte, 0, maxByteLen)

	// remaining bits in current byte
	rb := r.currBitIndex + 1
	var bitsToRead uint8
	if nBits <= rb {
		bitsToRead = nBits
	} else {
		bitsToRead = rb
	}

	tempByte := r.mustReadNBitsInCurrentByte(bitsToRead)
	tempByte = tempByte << (8 - bitsToRead) // left align
	tempBit := bitsToRead
	nBits -= bitsToRead

	if tempBit == 8 {
		result = append(result, tempByte)
		tempByte = 0
		tempBit = 0
	}

	for nBits >= 8 {
		err := r.fillBufIfNeeded()
		if err != nil {
			return nil, err
		}

		bitsToRead = 8
		b := r.mustReadNBitsInCurrentByte(bitsToRead)
		b1 := b >> tempBit
		b2 := b << (8 - tempBit)

		tempByte = tempByte | b1
		result = append(result, tempByte)
		tempByte = b2

		nBits -= 8
	}

	if nBits > 0 {
		err := r.fillBufIfNeeded()
		if err != nil {
			return nil, err
		}

		bitsToRead = nBits
		b := r.mustReadNBitsInCurrentByte(bitsToRead)
		b1 := b >> (bitsToRead - (8 - tempBit))       // wants to have (8 - tempBit) bits from b. b has bitsToRead bits
		b2 := b << (8 - (bitsToRead - (8 - tempBit))) // wants to have (bitsToRead - <bits of b1>) left aligned.

		tempByte = tempByte | b1
		result = append(result, tempByte)

		if padOne {
			b2 = b2 | (0xff >> tempBit)
		}
		result = append(result, b2)
	} else {
		if tempBit > 0 {
			result = append(result, tempByte)
		}
	}

	if alignRight {

	}

	return result, nil
}
