package bitstream_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/bearmini/bitstream-go"
)

func ExampleReader() {
	// binary expression:
	// 0000 0001 0010 0011 0100 0101 0110 0111 1000 1001 1010 1011 1100 1101 1110 1111
	data := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}

	// Reader
	r := bitstream.NewReader(bytes.NewReader(data), nil)

	// read a single bit
	bit0, err := r.ReadBit()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	fmt.Printf("bit: %1b\n", bit0)

	// read 2 bits
	bit1to2, err := r.ReadNBitsAsUint8(2)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	fmt.Printf("bits: %02b\n", bit1to2)

	// read 10 bits as big endian
	bit3to12, err := r.ReadNBitsAsUint16BE(10)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	fmt.Printf("bits: %010b\n", bit3to12)

	// read 20 bits as big endian
	bit13to32, err := r.ReadNBitsAsUint32BE(20)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	fmt.Printf("bits: %020b\n", bit13to32)

	// Output:
	// bit: 0
	// bits: 00
	// bits: 0000100100
	// bits: 01101000101011001111
}

func ExampleWriter() {
	dst := bytes.NewBuffer([]byte{})

	// Writer
	w := bitstream.NewWriter(dst)

	// Write a single bit `1`
	err := w.WriteBit(1)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	// Write a bool value as a bit (true: 1, false: 0)
	err = w.WriteBool(false)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	// Write 2 bits `10`
	err = w.WriteNBitsOfUint8(2, 0x02)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	// Write 8 bits `0101 0011`
	err = w.WriteUint8(0x53)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	// Write 10 bits `11 0010 1101`
	err = w.WriteNBitsOfUint16BE(10, 0x032d)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	// Write 16 bits `0000 1111 0101 1010`
	err = w.WriteUint16BE(0x0f5a)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	w.Flush()

	// we have written the following bits:
	// 1
	//  0
	//   10
	//      0101 0011
	//                1100 1011 01
	//                            00 0011 1101 0110 10
	// 1010 0101 0011 1100 1011 0100 0011 1101 0110 10xx

	fmt.Printf("%s", hex.EncodeToString(dst.Bytes()))
	// Output:
	// a53cb43d68
}
