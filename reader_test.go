package bitstream

import (
	"bytes"
	"crypto/rand"
	"reflect"
	"testing"
)

type indecies struct {
	BitIndex  uint8
	ByteIndex int
}

func TestForwardIndecies(t *testing.T) {
	testData := []struct {
		Name             string
		Data             []byte
		Start            indecies
		NumBitsToForward uint8
		End              indecies
	}{
		{
			Name:             "pattern 1",
			Data:             []byte{0x01},                        // b7654 3210
			Start:            indecies{BitIndex: 7, ByteIndex: 0}, //  ^
			NumBitsToForward: 1,
			End:              indecies{BitIndex: 6, ByteIndex: 0}, //   ^
		},
		{
			Name:             "pattern 2",
			Data:             []byte{0x02},                        // b7654 3210
			Start:            indecies{BitIndex: 6, ByteIndex: 0}, //   ^
			NumBitsToForward: 2,
			End:              indecies{BitIndex: 4, ByteIndex: 0}, //     ^
		},
		{
			Name:             "pattern 3",
			Data:             []byte{0x04},                        // b7654 3210
			Start:            indecies{BitIndex: 4, ByteIndex: 0}, //     ^
			NumBitsToForward: 4,
			End:              indecies{BitIndex: 0, ByteIndex: 0}, //          ^
		},
		{
			Name:             "pattern 5",
			Data:             []byte{0x05},                        // b7654 3210 |
			Start:            indecies{BitIndex: 0, ByteIndex: 0}, //          ^
			NumBitsToForward: 1,
			End:              indecies{BitIndex: 7, ByteIndex: 1}, //            ^
		},
		{
			Name:             "pattern 6",
			Data:             []byte{0x06, 0x06},                  // b7654 3210 | 7654 3210
			Start:            indecies{BitIndex: 0, ByteIndex: 0}, //          ^
			NumBitsToForward: 2,
			End:              indecies{BitIndex: 6, ByteIndex: 1}, //                ^
		},
		{
			Name:             "pattern 7",
			Data:             []byte{0x07, 0x07, 0x07},            // b7654 3210 | 7654 3210 | 7654 3210
			Start:            indecies{BitIndex: 1, ByteIndex: 0}, //         ^
			NumBitsToForward: 10,
			End:              indecies{BitIndex: 7, ByteIndex: 2}, //                          ^
		},
	}

	for _, data := range testData {
		data := data // capture
		t.Run(data.Name, func(t *testing.T) {
			//t.Parallel()

			r := NewReader(bytes.NewReader(data.Data), nil)
			r.fillBuf()
			r.currBitIndex = data.Start.BitIndex
			r.currByteIndex = data.Start.ByteIndex

			r.forwardIndecies(data.NumBitsToForward)

			if data.End.BitIndex != r.currBitIndex {
				t.Fatalf("\nunexpected bit index\nExpected: %+v\nActual:   %+v\n", data.End.BitIndex, r.currBitIndex)
			}
			if data.End.ByteIndex != r.currByteIndex {
				t.Fatalf("\nunexpected byte index\nExpected: %+v\nActual:   %+v\n", data.End.ByteIndex, r.currByteIndex)
			}
		})
	}
}

func TestReadBit(t *testing.T) {
	testData := []struct {
		Name         string
		Data         []byte
		ExpectedBits []byte
	}{
		{
			Name:         "pattern 1",
			Data:         []byte{0xaa},
			ExpectedBits: []byte{1, 0, 1, 0, 1, 0, 1, 0},
		},
		{
			Name:         "pattern 2",
			Data:         []byte{0x55, 0x12},
			ExpectedBits: []byte{0, 1, 0, 1, 0, 1, 0, 1, 0, 0, 0, 1, 0, 0, 1, 0},
		},
	}

	for _, data := range testData {
		data := data // capture
		t.Run(data.Name, func(t *testing.T) {
			//t.Parallel()

			r := NewReader(bytes.NewReader(data.Data), nil)
			for i, expectedBit := range data.ExpectedBits {
				actualBit, err := r.ReadBit()
				if err != nil {
					t.Fatalf("unexpected error: %+v\n", err)
				}
				if expectedBit != actualBit {
					t.Fatalf("\nbit %d\nExpected: %+v\nActual:   %+v\n", i, expectedBit, actualBit)
				}
			}

			_, err := r.ReadBit()
			if err == nil {
				t.Fatal("error should occur but no error\n")
			}
		})
	}
}

// https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go
var toEliminateCompilerOptimizationByte byte
var toEliminateCompilerOptimizationUint16 uint16
var toEliminateCompilerOptimizationUint32 uint32
var toEliminateCompilerOptimizationUint64 uint64

func BenchmarkReadBit(b *testing.B) {
	var v byte
	r := NewReader(rand.Reader, nil)
	for n := 0; n < b.N; n++ {
		v, _ = r.ReadBit()
	}
	toEliminateCompilerOptimizationByte = v
}

func TestReadNBitsAsUint8(t *testing.T) {
	testData := []struct {
		Name     string
		Data     []byte
		Start    indecies
		NBits    uint8
		Expected uint8
	}{
		{
			Name:     "pattern 1",                         // b7654 3210
			Data:     []byte{0x12},                        //  0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0}, //   ^
			NBits:    4,                                   //   ^^^ ^
			Expected: 2,                                   //   001 0 => 2
		},
		{
			Name:     "pattern 2",                         // b7654 3210
			Data:     []byte{0x12},                        //  0001 0010
			Start:    indecies{BitIndex: 7, ByteIndex: 0}, //  ^
			NBits:    8,                                   //  ^^^^ ^^^^
			Expected: 0x12,                                //  0001 0010 => 0x12
		},
		{
			Name:     "pattern 3",                         // b7654 3210
			Data:     []byte{0x12},                        //  0001 0010
			Start:    indecies{BitIndex: 0, ByteIndex: 0}, //          ^
			NBits:    1,                                   //          ^
			Expected: 0,                                   //          0 => 0
		},
		{
			Name:     "pattern 4",                         // b7654 3210 7654 3210
			Data:     []byte{0x12, 0x34},                  //  0001 0010 0011 0100
			Start:    indecies{BitIndex: 0, ByteIndex: 0}, //          ^
			NBits:    6,                                   //          ^ ^^^^ ^
			Expected: 6,                                   //          0 0011 0 => 6
		},
		{
			Name:     "pattern 5",                         // b7654 3210 7654 3210
			Data:     []byte{0xff, 0xff},                  //  1111 1111 1111 1111
			Start:    indecies{BitIndex: 3, ByteIndex: 0}, //       ^
			NBits:    8,                                   //       ^^^^ ^^^^
			Expected: 0xff,                                //       1111 1111 => 0xff
		},
		{
			Name:     "pattern 6",                         // b7654 3210 7654 3210
			Data:     []byte{0xaa, 0x55},                  //  1010 1010 0101 0101
			Start:    indecies{BitIndex: 2, ByteIndex: 0}, //        ^
			NBits:    7,                                   //        ^^^ ^^^^
			Expected: 0x25,                                //        010 0101 => 0x25
		},
	}

	for _, data := range testData {
		data := data // capture
		t.Run(data.Name, func(t *testing.T) {
			//t.Parallel()

			r := NewReader(bytes.NewReader(data.Data), nil)
			r.fillBuf()
			r.currBitIndex = data.Start.BitIndex
			r.currByteIndex = data.Start.ByteIndex

			v, err := r.ReadNBitsAsUint8(data.NBits)
			if err != nil {
				t.Fatalf("unexpected error: %+v\n", err)
			}
			if data.Expected != v {
				t.Fatalf("\nExpected: %+v\nActual:   %+v\n", data.Expected, v)
			}

		})
	}
}

func benchmarkReadNBitsAsUint8(b *testing.B, nBits uint8) {
	var v byte
	r := NewReader(rand.Reader, nil)
	for n := 0; n < b.N; n++ {
		v, _ = r.ReadNBitsAsUint8(nBits)
	}
	toEliminateCompilerOptimizationByte = v
}

func BenchmarkRead1BitAsUint8(b *testing.B) {
	benchmarkReadNBitsAsUint8(b, 1)
}

func BenchmarkRead2BitsAsUint8(b *testing.B) {
	benchmarkReadNBitsAsUint8(b, 2)
}

func BenchmarkRead3BitsAsUint8(b *testing.B) {
	benchmarkReadNBitsAsUint8(b, 3)
}

func BenchmarkRead4BitsAsUint8(b *testing.B) {
	benchmarkReadNBitsAsUint8(b, 4)
}

func BenchmarkRead5BitsAsUint8(b *testing.B) {
	benchmarkReadNBitsAsUint8(b, 5)
}

func BenchmarkRead6BitsAsUint8(b *testing.B) {
	benchmarkReadNBitsAsUint8(b, 6)
}

func BenchmarkRead7BitsAsUint8(b *testing.B) {
	benchmarkReadNBitsAsUint8(b, 7)
}

func BenchmarkRead8BitsAsUint8(b *testing.B) {
	benchmarkReadNBitsAsUint8(b, 8)
}

func TestReadNBitsAsUint16BE(t *testing.T) {
	testData := []struct {
		Name     string
		Data     []byte
		Start    indecies
		NBits    uint8
		Expected uint16
	}{
		{
			Name:     "pattern 1",                         // b7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56},            //  0001 0010 | 0011 0100 | 0101 0110
			Start:    indecies{BitIndex: 6, ByteIndex: 0}, //   ^
			NBits:    4,                                   //   ^^^ ^
			Expected: 2,                                   //   001 0 => 2
		},
		{
			Name:     "pattern 2",                         // b7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56},            //  0001 0010 | 0011 0100 | 0101 0110
			Start:    indecies{BitIndex: 6, ByteIndex: 0}, //   ^
			NBits:    9,                                   //   ^^^ ^^^^   ^^
			Expected: 0x48,                                //   001 0010   00 => 0x48
		},
		{
			Name:     "pattern 3",                         // b7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56},            //  0001 0010 | 0011 0100 | 0101 0110
			Start:    indecies{BitIndex: 6, ByteIndex: 0}, //   ^
			NBits:    16,                                  //   ^^^ ^^^^   ^^^^ ^^^^   ^
			Expected: 0x2468,                              //   001 0010   0011 0100   0 => 0x2468
		},
		{
			Name:     "pattern 4",                         // b7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56},            //  0001 0010 | 0011 0100 | 0101 0110
			Start:    indecies{BitIndex: 7, ByteIndex: 1}, //              ^
			NBits:    16,                                  //              ^^^^ ^^^^   ^^^^ ^^^^
			Expected: 0x3456,                              //              0011 0100   0101 0110   0 => 0x3456
		},
	}

	for _, data := range testData {
		data := data // capture
		t.Run(data.Name, func(t *testing.T) {
			//t.Parallel()

			r := NewReader(bytes.NewReader(data.Data), nil)
			r.fillBuf()
			r.currBitIndex = data.Start.BitIndex
			r.currByteIndex = data.Start.ByteIndex

			v, err := r.ReadNBitsAsUint16BE(data.NBits)
			if err != nil {
				t.Fatalf("unexpected error: %+v\n", err)
			}
			if data.Expected != v {
				t.Fatalf("\nExpected: %+v\nActual:   %+v\n", data.Expected, v)
			}

		})
	}
}

func benchmarkReadNBitsAsUint16BE(b *testing.B, nBits uint8) {
	var v uint16
	r := NewReader(rand.Reader, nil)
	for n := 0; n < b.N; n++ {
		v, _ = r.ReadNBitsAsUint16BE(nBits)
	}
	toEliminateCompilerOptimizationUint16 = v
}

func BenchmarkRead1BitAsUint16BE(b *testing.B) {
	benchmarkReadNBitsAsUint16BE(b, 1)
}

func BenchmarkRead2BitsAsUint16BE(b *testing.B) {
	benchmarkReadNBitsAsUint16BE(b, 2)
}

func BenchmarkRead9BitsAsUint16BE(b *testing.B) {
	benchmarkReadNBitsAsUint16BE(b, 9)
}

func BenchmarkRead10BitsAsUint16BE(b *testing.B) {
	benchmarkReadNBitsAsUint16BE(b, 10)
}

func BenchmarkRead11BitsAsUint16BE(b *testing.B) {
	benchmarkReadNBitsAsUint16BE(b, 11)
}

func BenchmarkRead12BitsAsUint16BE(b *testing.B) {
	benchmarkReadNBitsAsUint16BE(b, 12)
}

func BenchmarkRead13BitsAsUint16BE(b *testing.B) {
	benchmarkReadNBitsAsUint16BE(b, 13)
}

func BenchmarkRead14BitsAsUint16BE(b *testing.B) {
	benchmarkReadNBitsAsUint16BE(b, 14)
}

func BenchmarkRead15BitsAsUint16BE(b *testing.B) {
	benchmarkReadNBitsAsUint16BE(b, 15)
}

func BenchmarkRead16BitsAsUint16BE(b *testing.B) {
	benchmarkReadNBitsAsUint16BE(b, 16)
}

func TestReadNBitsAsUint32BE(t *testing.T) {
	testData := []struct {
		Name     string
		Data     []byte
		Start    indecies
		NBits    uint8
		Expected uint32
	}{
		{
			Name:     "pattern 1",                          // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},  //   ^
			NBits:    4,                                    //   ^^^ ^
			Expected: 2,                                    //   001 0 => 2
		},
		{
			Name:     "pattern 2",                          // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},  //   ^
			NBits:    9,                                    //   ^^^ ^^^^   ^^
			Expected: 0x48,                                 //   001 0010   00 => 0x48
		},
		{
			Name:     "pattern 3",                          // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},  //   ^
			NBits:    17,                                   //   ^^^ ^^^^   ^^^^ ^^^^   ^^
			Expected: 0x48D1,                               //   001 0010   0011 0100   01 => 0x48D1
		},
		{
			Name:     "pattern 4",                          // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},  //   ^
			NBits:    24,                                   //   ^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^
			Expected: 0x2468AC,                             //   001 0010   0011 0100   0101 0110   0 => 0010 0100 0110 1000 1010 1100 => 0x2468AC
		},
		{
			Name:     "pattern 5",                          // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},  //   ^
			NBits:    32,                                   //   ^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^
			Expected: 0x2468ACF1,                           //   001 0010   0011 0100   0101 0110   0111 1000   1 => 0010 0100 0110 1000 1010 1100 1111 0001 => 0x2468ACF1
		},
		{
			Name:     "pattern 6",                          // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010
			Start:    indecies{BitIndex: 7, ByteIndex: 1},  //              ^
			NBits:    32,                                   //              ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^
			Expected: 0x3456789A,                           //              0011 0100   0101 0110   0111 1000   1001 1010 => 0x3456789A
		},
	}

	for _, data := range testData {
		data := data // capture
		t.Run(data.Name, func(t *testing.T) {
			//t.Parallel()

			r := NewReader(bytes.NewReader(data.Data), nil)
			r.fillBuf()
			r.currBitIndex = data.Start.BitIndex
			r.currByteIndex = data.Start.ByteIndex

			v, err := r.ReadNBitsAsUint32BE(data.NBits)
			if err != nil {
				t.Fatalf("unexpected error: %+v\n", err)
			}
			if data.Expected != v {
				t.Fatalf("\nExpected: %+v\nActual:   %+v\n", data.Expected, v)
			}

		})
	}
}

func benchmarkReadNBitsAsUint32BE(b *testing.B, nBits uint8) {
	var v uint32
	r := NewReader(rand.Reader, nil)
	for n := 0; n < b.N; n++ {
		v, _ = r.ReadNBitsAsUint32BE(nBits)
	}
	toEliminateCompilerOptimizationUint32 = v
}

func BenchmarkRead1BitAsUint32BE(b *testing.B) {
	benchmarkReadNBitsAsUint32BE(b, 1)
}

func BenchmarkRead2BitsAsUint32BE(b *testing.B) {
	benchmarkReadNBitsAsUint32BE(b, 2)
}

func BenchmarkRead9BitsAsUint32BE(b *testing.B) {
	benchmarkReadNBitsAsUint32BE(b, 9)
}

func BenchmarkRead10BitsAsUint32BE(b *testing.B) {
	benchmarkReadNBitsAsUint32BE(b, 10)
}

func BenchmarkRead11BitsAsUint32BE(b *testing.B) {
	benchmarkReadNBitsAsUint32BE(b, 17)
}

func BenchmarkRead12BitsAsUint32BE(b *testing.B) {
	benchmarkReadNBitsAsUint32BE(b, 18)
}
func BenchmarkRead15BitsAsUint32BE(b *testing.B) {
	benchmarkReadNBitsAsUint32BE(b, 31)
}

func BenchmarkRead32BitsAsUint32BE(b *testing.B) {
	benchmarkReadNBitsAsUint32BE(b, 32)
}

func TestReadNBitsAsUint64BE(t *testing.T) {
	testData := []struct {
		Name     string
		Data     []byte
		Start    indecies
		NBits    uint8
		Expected uint64
	}{
		{
			Name:     "pattern 1",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},                          //   ^
			NBits:    4,                                                            //   ^^^ ^
			Expected: 2,                                                            //   001 0 => 2
		},
		{
			Name:     "pattern 2",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},                          //   ^
			NBits:    9,                                                            //   ^^^ ^^^^   ^^
			Expected: 0x48,                                                         //   001 0010   00 => 0x48
		},
		{
			Name:     "pattern 3",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},                          //   ^
			NBits:    17,                                                           //   ^^^ ^^^^   ^^^^ ^^^^   ^^
			Expected: 0x48D1,                                                       //   001 0010   0011 0100   01 => 0x48D1
		},
		{
			Name:     "pattern 4",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},                          //   ^
			NBits:    33,                                                           //   ^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^
			Expected: 0x48D159E2,                                                   //   001 0010   0011 0100   0101 0110   0111 1000   10 => 0 0100 1000 1101 0001 0101 1001 1110 0010 => 0x48D159E2
		},
		{
			Name:     "pattern 5",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},                          //   ^
			NBits:    42,                                                           //   ^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^
			Expected: 0x91A2B3C4D5,                                                 //   001 0010   0011 0100   0101 0110   0111 1000   1001 1010   101 => 00 1001 0001 1010 0010 1011 0011 1100 0100 1101 0101 => 0x91A2B3C4D5
		},
		{
			Name:     "pattern 6",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},                          //   ^
			NBits:    51,                                                           //   ^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^
			Expected: 0x123456789ABCD,                                              //   001 0010   0011 0100   0101 0110   0111 1000   1001 1010   1011 1100   1101 => 001 0010 0011 0100 0101 0110 0111 1000 1001 1010 1011 1100 1101 => 0x123456789ABCD
		},
		{
			Name:     "pattern 7",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},                          //   ^
			NBits:    60,                                                           //   ^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^
			Expected: 0x2468ACF13579BDE,                                            //   001 0010   0011 0100   0101 0110   0111 1000   1001 1010   1011 1100   1101 1110   1111 0 => 0010 0100 0110 1000 1010 1100 1111 0001 0011 0101 0111 1001 1011 1101 1110 => 0x2468ACF13578BDE
		},
		{
			Name:     "pattern 6",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 7, ByteIndex: 1},                          //              ^
			NBits:    64,                                                           //              ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^
			Expected: 0x3456789ABCDEF012,                                           //              0011 0100   0101 0110   0111 1000   1001 1010   1011 1100   1101 1110   1111 0000   0001 0010 => 0x3456789ABCDEF012
		},
	}

	for _, data := range testData {
		data := data // capture
		t.Run(data.Name, func(t *testing.T) {
			//t.Parallel()

			r := NewReader(bytes.NewReader(data.Data), nil)
			r.fillBuf()
			r.currBitIndex = data.Start.BitIndex
			r.currByteIndex = data.Start.ByteIndex

			v, err := r.ReadNBitsAsUint64BE(data.NBits)
			if err != nil {
				t.Fatalf("unexpected error: %+v\n", err)
			}
			if data.Expected != v {
				t.Fatalf("\nExpected: %+v\nActual:   %+v\n", data.Expected, v)
			}

		})
	}
}

func benchmarkReadNBitsAsUint64BE(b *testing.B, nBits uint8) {
	var v uint64
	r := NewReader(rand.Reader, nil)
	for n := 0; n < b.N; n++ {
		v, _ = r.ReadNBitsAsUint64BE(nBits)
	}
	toEliminateCompilerOptimizationUint64 = v
}

func BenchmarkRead1BitAsUint64BE(b *testing.B) {
	benchmarkReadNBitsAsUint64BE(b, 1)
}

func BenchmarkRead2BitsAsUint64BE(b *testing.B) {
	benchmarkReadNBitsAsUint64BE(b, 2)
}

func BenchmarkRead9BitsAsUint64BE(b *testing.B) {
	benchmarkReadNBitsAsUint64BE(b, 9)
}

func BenchmarkRead10BitsAsUint64BE(b *testing.B) {
	benchmarkReadNBitsAsUint64BE(b, 10)
}

func BenchmarkRead11BitsAsUint64BE(b *testing.B) {
	benchmarkReadNBitsAsUint64BE(b, 17)
}

func BenchmarkRead12BitsAsUint64BE(b *testing.B) {
	benchmarkReadNBitsAsUint64BE(b, 18)
}
func BenchmarkRead15BitsAsUint64BE(b *testing.B) {
	benchmarkReadNBitsAsUint64BE(b, 31)
}

func BenchmarkRead64BitsAsUint64BE(b *testing.B) {
	benchmarkReadNBitsAsUint64BE(b, 64)
}

func TestReadNBits(t *testing.T) {
	testData := []struct {
		Name       string
		Data       []byte
		Start      indecies
		NBits      uint8
		AlignRight bool
		PadOne     bool
		Expected   []byte
	}{
		{
			Name:     "pattern 1",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},                          //   ^
			NBits:    4,                                                            //   ^^^ ^
			Expected: []byte{0x20},                                                 //   001 0  => 0010 0000 => 0x20
		},
		{
			Name:     "pattern 2",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},                          //   ^
			NBits:    8,                                                            //   ^^^ ^^^^   ^
			Expected: []byte{0x24},                                                 //   001 0010   0 => 00100100 => 0x24
		},
		{
			Name:     "pattern 3",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},                          //   ^
			NBits:    9,                                                            //   ^^^ ^^^^   ^^
			Expected: []byte{0x24, 0x00},                                           //   001 0010   00 => 00100100 0 => 0x24 0x00
		},
		{
			Name:     "pattern 4",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},                          //   ^
			NBits:    16,                                                           //   ^^^ ^^^^   ^^^^ ^^^^   ^
			Expected: []byte{0x24, 0x68},                                           //   001 0010   0011 0100   0 => 0010 0100 0110 1000 => 0x24 0x68
		},
		{
			Name:     "pattern 5",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},                          //   ^
			NBits:    17,                                                           //   ^^^ ^^^^   ^^^^ ^^^^   ^^
			Expected: []byte{0x24, 0x68, 0x80},                                     //   001 0010   0011 0100   01 => 0010 0100 0110 1000 1 => 0x24 0x68 0x80
		},
		{
			Name:     "pattern 6",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},                          //   ^
			NBits:    33,                                                           //   ^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^
			Expected: []byte{0x24, 0x68, 0xac, 0xf1, 0x00},                         //   001 0010   0011 0100   0101 0110   0111 1000   10 => 0010 0100 0110 1000 1010 1100 1111 0001 0 => 0x24 0x68 0xac 0xf1 0x00
		},
		{
			Name:     "pattern 7",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},                          //   ^
			NBits:    42,                                                           //   ^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^
			Expected: []byte{0x24, 0x68, 0xac, 0xf1, 0x35, 0x40},                   //   001 0010   0011 0100   0101 0110   0111 1000   1001 1010   101 => 0010 0100 0110 1000 1010 1100 1111 0001 0011 0101 01 => 0x24 0x68 0xac 0xf1 0x35 0x40
		},
		{
			Name:     "pattern 8",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},                          //   ^
			NBits:    51,                                                           //   ^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^
			Expected: []byte{0x24, 0x68, 0xac, 0xf1, 0x35, 0x79, 0xa0},             //   001 0010   0011 0100   0101 0110   0111 1000   1001 1010   1011 1100   1101 => 0010 0100 0110 1000 1010 1100 1111 0001 0011 0101 0111 1001 101 => 0x24 0x68 0xac 0xf1 0x35 0x79 0xa0
		},
		{
			Name:     "pattern 9",                                                  // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 6, ByteIndex: 0},                          //   ^
			NBits:    60,                                                           //   ^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^
			Expected: []byte{0x24, 0x68, 0xac, 0xf1, 0x35, 0x79, 0xbd, 0xe0},       //   001 0010   0011 0100   0101 0110   0111 1000   1001 1010   1011 1100   1101 1110   1111 0 => 0010 0100 0110 1000 1010 1100 1111 0001 0011 0101 0111 1001 1011 1101 1110 => 0x24 0x68 0xAC 0xF1 0x35 0x79 0xBD 0xE0
		},
		{
			Name:     "pattern 10",                                                 // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 7, ByteIndex: 1},                          //              ^
			NBits:    64,                                                           //              ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^   ^^^^ ^^^^
			Expected: []byte{0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0, 0x12},       //              0011 0100   0101 0110   0111 1000   1001 1010   1011 1100   1101 1110   1111 0000   0001 0010 => 0x34 56 78 9A BC DE F0 12
		},
		{
			Name:     "pattern 11",                                                 // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 5, ByteIndex: 0},                          //    ^
			NBits:    17,                                                           //    ^^ ^^^^   ^^^^ ^^^^   ^^^
			Expected: []byte{0x48, 0xd1, 0x00},                                     //    01 0010   0011 0100   010 => 0100 1000 1101 0001 0 => 0x48 0xD1 0x00
		},
		{
			Name:     "pattern 12",                                                 // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 4, ByteIndex: 0},                          //     ^
			NBits:    17,                                                           //     ^ ^^^^   ^^^^ ^^^^   ^^^^
			Expected: []byte{0x91, 0xa2, 0x80},                                     //     1 0010   0011 0100   0101 => 1001 0001 1010 0010 1 => 0x91 0xa2 0x80
		},
		{
			Name:     "pattern 13",                                                 // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 3, ByteIndex: 0},                          //       ^
			NBits:    17,                                                           //       ^^^^   ^^^^ ^^^^   ^^^^ ^
			Expected: []byte{0x23, 0x45, 0x00},                                     //       0010   0011 0100   0101 0 => 0010 0011 0100 0101 0 => 0x23 0x45 0x00
		},
		{
			Name:     "pattern 14",                                                 // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 2, ByteIndex: 0},                          //        ^
			NBits:    17,                                                           //        ^^^   ^^^^ ^^^^   ^^^^ ^^
			Expected: []byte{0x46, 0x8a, 0x80},                                     //        010   0011 0100   0101 01 => 0100 0110 1000 1010 1 => 0x46 0x8a 0x80
		},
		{
			Name:     "pattern 15",                                                 // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 1, ByteIndex: 0},                          //         ^
			NBits:    17,                                                           //         ^^   ^^^^ ^^^^   ^^^^ ^^^
			Expected: []byte{0x8d, 0x15, 0x80},                                     //         10   0011 0100   0101 011 => 1000 1101 0001 0101 1 => 0x8D 0x15 0x80
		},
		{
			Name:     "pattern 16",                                                 // b7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210 | 7654 3210
			Data:     []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12}, //  0001 0010 | 0011 0100 | 0101 0110 | 0111 1000 | 1001 1010 | 1011 1100 | 1101 1110 | 1111 0000 | 0001 0010
			Start:    indecies{BitIndex: 0, ByteIndex: 0},                          //          ^
			NBits:    17,                                                           //          ^   ^^^^ ^^^^   ^^^^ ^^^^
			Expected: []byte{0x1a, 0x2b, 0x00},                                     //          0   0011 0100   0101 0110 => 0001 1010 0010 1011 0 => 0x1A 0x2B 0x00
		},
	}

	for _, data := range testData {
		data := data // capture
		t.Run(data.Name, func(t *testing.T) {
			//t.Parallel()

			r := NewReader(bytes.NewReader(data.Data), nil)
			r.fillBuf()
			r.currBitIndex = data.Start.BitIndex
			r.currByteIndex = data.Start.ByteIndex

			v, err := r.ReadNBits(data.NBits, &ReadOptions{AlignRight: data.AlignRight, PadOne: data.PadOne})
			if err != nil {
				t.Fatalf("unexpected error: %+v\n", err)
			}
			if !reflect.DeepEqual(data.Expected, v) {
				t.Fatalf("\nExpected: %+v\nActual:   %+v\n", data.Expected, v)
			}

		})
	}
}

func benchmarkReadNBits(b *testing.B, nBits uint8) {
	var v uint64
	r := NewReader(rand.Reader, nil)
	for n := 0; n < b.N; n++ {
		v, _ = r.ReadNBitsAsUint64BE(nBits)
	}
	toEliminateCompilerOptimizationUint64 = v
}

func BenchmarkRead1Bit(b *testing.B) {
	benchmarkReadNBits(b, 1)
}

func BenchmarkRead2Bits(b *testing.B) {
	benchmarkReadNBits(b, 2)
}

func BenchmarkRead9Bits(b *testing.B) {
	benchmarkReadNBits(b, 9)
}

func BenchmarkRead10Bits(b *testing.B) {
	benchmarkReadNBits(b, 10)
}

func BenchmarkRead11Bits(b *testing.B) {
	benchmarkReadNBits(b, 17)
}

func BenchmarkRead12Bits(b *testing.B) {
	benchmarkReadNBits(b, 18)
}
func BenchmarkRead15Bits(b *testing.B) {
	benchmarkReadNBits(b, 31)
}

func BenchmarkRead64Bits(b *testing.B) {
	benchmarkReadNBits(b, 64)
}
