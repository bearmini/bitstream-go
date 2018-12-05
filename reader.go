package bitstream

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

const (
	DefaultBufferSize = 1024
)

// Reader is a bit stream reader.
// It does not have io.Reader interface.
type Reader struct {
	src           io.Reader
	srcEOF        bool
	buf           []byte
	bufLen        int
	currByteIndex int   // starts from 0
	currBitIndex  uint8 // MSB: 7, LSB: 0
	opt           *ReaderOptions
}

// ReaderOptions is a set of options for creating a Reader.
type ReaderOptions struct {
	BufferSize uint
}

// GetBufferSize gets configured buffer size.
func (opt *ReaderOptions) GetBufferSize() uint {
	if opt == nil || opt.BufferSize == 0 {
		return DefaultBufferSize
	}
	return opt.BufferSize
}

// NewReader creates a new Reader instance with options.
func NewReader(src io.Reader, opt *ReaderOptions) *Reader {
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

// ReadBit reads a single bit from the bit stream.
// The bit read from the stream will be set in the LSB of the return value.
func (r *Reader) ReadBit() (byte, error) {
	err := r.fillBufIfNeeded()
	if err != nil {
		return 0, err
	}

	b := r.buf[r.currByteIndex]
	mask := uint8(1 << r.currBitIndex)
	result := (b & mask) >> r.currBitIndex
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
	mask := uint8((1 << (r.currBitIndex + 1)) - 1)
	result := (b & mask) >> (r.currBitIndex - (nBits - 1))
	r.forwardIndecies(nBits)
	return result
}

// ReadNBitsAsUint8 reads `nBits` bits as a unsigned integer from the bit stream and returns it in uint8 (LSB aligned).
// `nBits` must be less than or equal to 8, otherwise returns an error.
// If `nBits` == 0, this function always returns 0.
func (r *Reader) ReadNBitsAsUint8(nBits uint8) (uint8, error) {
	if nBits == 0 {
		return 0, nil
	}

	if nBits > 8 {
		return 0, errors.New("nBits too large for uint8")
	}

	err := r.fillBufIfNeeded()
	if err != nil {
		return 0, err
	}

	// remaining bits in current byte
	rb := r.currBitIndex + 1

	if nBits <= rb { // can be read from the current byte
		b := r.mustReadNBitsInCurrentByte(nBits)
		return b, nil
	}

	// 8 bits are distributed in 2 bytes
	nBits1 := rb
	nBits2 := nBits - rb

	b1 := r.mustReadNBitsInCurrentByte(nBits1)
	b2, err := r.ReadNBitsAsUint8(nBits2)
	if err != nil {
		return 0, err
	}

	return (b1 << nBits2) | b2, nil
}

// ReadUint8 reads 8 bits from the bit stream and returns it in uint8.
func (r *Reader) ReadUint8() (uint8, error) {
	return r.ReadNBitsAsUint8(8)
}

// ReadNBitsAsUint16BE reads `nBits` bits as a big endian unsigned integer from the bit stream and returns it in uint16 (LSB aligned).
// `nBits` must be less than or equal to 16, otherwise returns an error.
// If `nBits` == 0, this function always returns 0.
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

	// 16 bits may be distributed in up to 3 bytes
	nBits1 := rb         // count of bits in the first byte
	nBits2 := nBits - rb // count of bits in the second byte
	nBits3 := uint8(0)   // count of bits in the third byte
	if nBits2 > 8 {
		nBits3 = nBits2 - 8
		nBits2 = 8
	}

	b1 := r.mustReadNBitsInCurrentByte(nBits1)
	b2, err := r.ReadNBitsAsUint8(nBits2)
	if err != nil {
		return 0, err
	}
	b3, err := r.ReadNBitsAsUint8(nBits3) // expects this function returns 0 if nBits3 == 0
	if err != nil {
		return 0, err
	}

	return (uint16(b1) << (nBits2 + nBits3)) | (uint16(b2) << nBits3) | uint16(b3), nil
}

// ReadUint16BE reads 16 bits as a big endian unsigned integer from the bit stream and returns it in uint16.
func (r *Reader) ReadUint16BE() (uint16, error) {
	return r.ReadNBitsAsUint16BE(16)
}

// ReadNBitsAsUint32BE reads `nBits` bits as a big endian unsigned integer from the bit stream and returns it in uint32 (LSB aligned).
// `nBits` must be less than or equal to 32, otherwise returns an error.
// If `nBits` == 0, this function always returns 0.
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

	// 32 bits may be distributed in up to 5 bytes
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

// ReadUint32BE reads 32 bits as a big endian unsigned integer from the bit stream and returns it in uint32.
func (r *Reader) ReadUint32BE() (uint32, error) {
	return r.ReadNBitsAsUint32BE(32)
}

// ReadNBitsAsInt32BE reads `nBits` bits as a big endian signed integer from the bit stream and returns it in int32 (LSB aligned).
// MSB is a sign bit.
// `nBits` must be less than or equal to 32, otherwise returns an error.
// If `nBits` == 0, this function always returns 0.
func (r *Reader) ReadNBitsAsInt32BE(nBits uint8) (int32, error) {
	v, err := r.ReadNBitsAsUint32BE(nBits)
	if err != nil {
		return 0, err
	}

	//fmt.Printf("v   == %#08x\n", v)
	msb := uint32(1) << (nBits - 1)
	//fmt.Printf("msb == %#08x\n", msb)

	if (v & msb) == 0 {
		return int32(v), nil
	}

	f := 0xffffffff & ^(msb - 1)
	//fmt.Printf("f   ==%#08x\n", f)
	//fmt.Printf("f|v ==%#08x\n", f|v)
	return int32(f | v), nil
}

// ReadNBitsAsUint64BE reads `nBits` bits as a big endian unsigned integer from the bit stream and returns it in uint64 (LSB aligned).
// `nBits` must be less than or equal to 64, otherwise returns an error.
// If `nBits` == 0, this function always returns 0.
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

	// 64bit value may be distributed in 9 bytes
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

// ReadUint64BE reads 64 bits as a big endian unsigned integer from the bit stream and returns it in uint64.
func (r *Reader) ReadUint64BE() (uint64, error) {
	return r.ReadNBitsAsUint64BE(64)
}

// ReadOptions is a set of options to read bits from the bit stream.
type ReadOptions struct {
	AlignRight bool // If true, returned value will be aligned to right (default: align to left)
	PadOne     bool // If true, returned value will be padded with '1' instead of '0' (default: pad with '0')
}

// ReadNBits reads `nBits` bits from the bit stream and returns it as a slice of bytes.
// If `nBits` == 0, this function always returns nil.
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

		if nBits > (8 - tempBit) {
			if padOne {
				b2 = b2 | (0xff >> tempBit)
			}
			result = append(result, b2)
		}
	} else {
		if tempBit > 0 {
			result = append(result, tempByte)
		}
	}

	if alignRight {
		return nil, errors.New("not implemented yet")
	}

	return result, nil
}
