package bitstream

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func TestWriteBit(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})
	bw := NewWriter(buf)

	bw.WriteBit(0)
	bw.WriteBit(1)
	bw.WriteBit(0)
	bw.WriteBit(1)
	bw.WriteBit(0)
	bw.WriteBit(0)
	bw.WriteBit(1)
	bw.WriteBit(1)
	bw.WriteBit(1)
	bw.WriteBit(0)
	bw.WriteBit(1)
	bw.WriteBit(0)
	bw.WriteBit(1)
	bw.WriteBit(1)
	bw.WriteBit(0)
	bw.WriteBit(0)

	expected := []byte{0x53, 0xac}
	if !reflect.DeepEqual(buf.Bytes(), expected) {
		t.Fatalf("\nExpected: %+v\nActual:   %+v\n", expected, buf.Bytes())
	}
	if uint(16) != bw.WrittenBits() {
		t.Fatalf("\nunexpected writtenBits\nExpected: %+v\nActual:   %+v\n", 16, bw.WrittenBits())
	}
}

func BenchmarkWriteBit(b *testing.B) {
	rand.Seed(time.Now().UnixNano())
	buf := bytes.NewBuffer([]byte{})
	bw := NewWriter(buf)
	for n := 0; n < b.N; n++ {
		_ = bw.WriteBit(uint8(rand.Intn(256)))
	}
}

type writerStatus struct {
	currByte     byte
	currBitIndex uint8
	buf          []byte
}

func TestWriteNBitsOfUint8(t *testing.T) {
	testData := []struct {
		Name     string
		NBits    uint8
		Value    uint8
		Start    writerStatus
		Expected writerStatus
	}{
		{
			Name:     "pattern 1",
			NBits:    1,
			Value:    0xff, // 1
			Start:    writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{}},
			Expected: writerStatus{currByte: 0x80, currBitIndex: 6, buf: []byte{}},
		},
		{
			Name:     "pattern 2",
			NBits:    4,
			Value:    0x02,                                                         // 0010
			Start:    writerStatus{currByte: 0x40, currBitIndex: 5, buf: []byte{}}, // 01xx xxxx
			Expected: writerStatus{currByte: 0x48, currBitIndex: 1, buf: []byte{}}, // 0100 10xx
		},
		{
			Name:     "pattern 3",
			NBits:    8,
			Value:    0xff,
			Start:    writerStatus{currByte: 0x00, currBitIndex: 3, buf: []byte{}},
			Expected: writerStatus{currByte: 0xf0, currBitIndex: 3, buf: []byte{0x0f}},
		},
		{
			Name:     "pattern 4",
			NBits:    8,
			Value:    0xaa,
			Start:    writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{}},
			Expected: writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{0xaa}},
		},
	}

	for _, data := range testData {
		data := data // capture
		t.Run(data.Name, func(t *testing.T) {
			//t.Parallel()

			buf := bytes.NewBuffer(data.Start.buf)
			bw := NewWriter(buf)

			bw.currByte[0] = data.Start.currByte
			bw.currBitIndex = data.Start.currBitIndex

			err := bw.WriteNBitsOfUint8(data.NBits, data.Value)
			if err != nil {
				t.Fatalf("unexpected error: %+v\n", err)
			}
			if uint(data.NBits) != bw.WrittenBits() {
				t.Fatalf("\nunexpected writtenBits\nExpected: %+v\nActual:   %+v\n", data.NBits, bw.WrittenBits())
			}
			if data.Expected.currByte != bw.currByte[0] {
				t.Fatalf("\nunexpected currByte\nExpected: %+v\nActual:   %+v\n", data.Expected.currByte, bw.currByte[0])
			}
			if data.Expected.currBitIndex != bw.currBitIndex {
				t.Fatalf("\nunexpected currBitIndex\nExpected: %+v\nActual:   %+v\n", data.Expected.currBitIndex, bw.currBitIndex)
			}
			if !reflect.DeepEqual(data.Expected.buf, buf.Bytes()) {
				t.Fatalf("\nunexpected flushed data\nExpected: %+v\nActual:   %+v\n", data.Expected.buf, buf.Bytes())
			}

		})
	}

}

func benchmarkWriteNBitsOfUint8(nBits uint8, b *testing.B) {
	rand.Seed(time.Now().UnixNano())
	buf := bytes.NewBuffer([]byte{})
	bw := NewWriter(buf)
	for n := 0; n < b.N; n++ {
		_ = bw.WriteNBitsOfUint8(nBits, uint8(rand.Intn(256)))
	}
}

func BenchmarkWrite1BitsOfUint8(b *testing.B) {
	benchmarkWriteNBitsOfUint8(1, b)
}

func BenchmarkWrite2BitsOfUint8(b *testing.B) {
	benchmarkWriteNBitsOfUint8(2, b)
}

func BenchmarkWrite3BitsOfUint8(b *testing.B) {
	benchmarkWriteNBitsOfUint8(3, b)
}

func BenchmarkWrite4BitsOfUint8(b *testing.B) {
	benchmarkWriteNBitsOfUint8(4, b)
}

func BenchmarkWrite5BitsOfUint8(b *testing.B) {
	benchmarkWriteNBitsOfUint8(5, b)
}

func BenchmarkWrite6BitsOfUint8(b *testing.B) {
	benchmarkWriteNBitsOfUint8(6, b)
}

func BenchmarkWrite7BitsOfUint8(b *testing.B) {
	benchmarkWriteNBitsOfUint8(7, b)
}

func BenchmarkWrite8BitsOfUint8(b *testing.B) {
	benchmarkWriteNBitsOfUint8(8, b)
}

func TestWriteNBitsOfUint16(t *testing.T) {
	testData := []struct {
		Name     string
		NBits    uint8
		Value    uint16
		Start    writerStatus
		Expected writerStatus
	}{
		{
			Name:     "pattern 1",
			NBits:    1,
			Value:    0xffff, // 1
			Start:    writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{}},
			Expected: writerStatus{currByte: 0x80, currBitIndex: 6, buf: []byte{}},
		},
		{
			Name:     "pattern 2",
			NBits:    10,
			Value:    0x2222,                                                           //   10 0010 0010
			Start:    writerStatus{currByte: 0x40, currBitIndex: 5, buf: []byte{}},     // 01xx xxxx
			Expected: writerStatus{currByte: 0x20, currBitIndex: 3, buf: []byte{0x62}}, // 0110 0010 0010 xxxx
		},
		{
			Name:     "pattern 3",
			NBits:    13,
			Value:    0xffff,                                                                 //      1111 1111 1111 1
			Start:    writerStatus{currByte: 0x00, currBitIndex: 3, buf: []byte{}},           // 0000 xxxx
			Expected: writerStatus{currByte: 0x80, currBitIndex: 6, buf: []byte{0x0f, 0xff}}, // 0000 1111 1111 1111 1xxx
		},
		{
			Name:     "pattern 4",
			NBits:    16,
			Value:    0xabcd,                                                                 // 1010 1011 1100 1101
			Start:    writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{}},           // xxxx xxxx
			Expected: writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{0xab, 0xcd}}, // 1010 1011 1100 1101
		},
		{
			Name:     "pattern 5",
			NBits:    16,
			Value:    0xabcd,                                                                 //        10 1010 1111 0011 01
			Start:    writerStatus{currByte: 0x88, currBitIndex: 1, buf: []byte{}},           // 1000 10xx
			Expected: writerStatus{currByte: 0x34, currBitIndex: 1, buf: []byte{0x8a, 0xaf}}, // 1000 1010 1010 1111 0011 01xx
		},
	}

	for _, data := range testData {
		data := data // capture
		t.Run(data.Name, func(t *testing.T) {
			//t.Parallel()

			buf := bytes.NewBuffer(data.Start.buf)
			bw := NewWriter(buf)

			bw.currByte[0] = data.Start.currByte
			bw.currBitIndex = data.Start.currBitIndex

			err := bw.WriteNBitsOfUint16(data.NBits, data.Value)
			if err != nil {
				t.Fatalf("unexpected error: %+v\n", err)
			}
			if uint(data.NBits) != bw.WrittenBits() {
				t.Fatalf("\nunexpected writtenBits\nExpected: %+v\nActual:   %+v\n", data.NBits, bw.WrittenBits())
			}
			if data.Expected.currByte != bw.currByte[0] {
				t.Fatalf("\nunexpected currByte\nExpected: %+v\nActual:   %+v\n", data.Expected.currByte, bw.currByte[0])
			}
			if data.Expected.currBitIndex != bw.currBitIndex {
				t.Fatalf("\nunexpected currBitIndex\nExpected: %+v\nActual:   %+v\n", data.Expected.currBitIndex, bw.currBitIndex)
			}
			if !reflect.DeepEqual(data.Expected.buf, buf.Bytes()) {
				t.Fatalf("\nunexpected flushed data\nExpected: %+v\nActual:   %+v\n", data.Expected.buf, buf.Bytes())
			}

		})
	}

}

func benchmarkWriteNBitsOfUint16(nBits uint8, b *testing.B) {
	rand.Seed(time.Now().UnixNano())
	buf := bytes.NewBuffer([]byte{})
	bw := NewWriter(buf)
	for n := 0; n < b.N; n++ {
		_ = bw.WriteNBitsOfUint16(nBits, uint16(rand.Intn(65536)))
	}
}

func BenchmarkWrite1BitsOfUint16(b *testing.B) {
	benchmarkWriteNBitsOfUint16(1, b)
}

func BenchmarkWrite2BitsOfUint16(b *testing.B) {
	benchmarkWriteNBitsOfUint16(2, b)
}

func BenchmarkWrite9BitsOfUint16(b *testing.B) {
	benchmarkWriteNBitsOfUint16(9, b)
}

func BenchmarkWrite10BitsOfUint16(b *testing.B) {
	benchmarkWriteNBitsOfUint16(10, b)
}

func BenchmarkWrite15BitsOfUint16(b *testing.B) {
	benchmarkWriteNBitsOfUint16(15, b)
}

func BenchmarkWrite16BitsOfUint16(b *testing.B) {
	benchmarkWriteNBitsOfUint16(16, b)
}

func TestWriteNBitsOfUint32(t *testing.T) {
	testData := []struct {
		Name     string
		NBits    uint8
		Value    uint32
		Start    writerStatus
		Expected writerStatus
	}{
		{
			Name:     "pattern 1",
			NBits:    1,
			Value:    0xffff, // 1
			Start:    writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{}},
			Expected: writerStatus{currByte: 0x80, currBitIndex: 6, buf: []byte{}},
		},
		{
			Name:     "pattern 2",
			NBits:    16,
			Value:    0xffffabcd,                                                             //        10 1010 1111 0011 01
			Start:    writerStatus{currByte: 0x88, currBitIndex: 1, buf: []byte{}},           // 1000 10xx
			Expected: writerStatus{currByte: 0x34, currBitIndex: 1, buf: []byte{0x8a, 0xaf}}, // 1000 1010 1010 1111 0011 01xx
		},
		{
			Name:     "pattern 3",
			NBits:    17,
			Value:    0xffffabcd,                                                             //        11 0101 0111 1001 101
			Start:    writerStatus{currByte: 0x88, currBitIndex: 1, buf: []byte{}},           // 1000 10xx
			Expected: writerStatus{currByte: 0x9a, currBitIndex: 0, buf: []byte{0x8b, 0x57}}, // 1000 1011 0101 0111 1001 101x
		},
		{
			Name:     "pattern 4",
			NBits:    24,
			Value:    0xffabcdef,                                                                   // 1010 1011 1100 1101 1110 1111
			Start:    writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{}},                 // xxxx xxxx
			Expected: writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{0xab, 0xcd, 0xef}}, // 1010 1011 1100 1101 1110 1111
		},
		{
			Name:     "pattern 5",
			NBits:    24,
			Value:    0xffabcdef,                                                                   //      1010 1011 1100 1101 1110 1111
			Start:    writerStatus{currByte: 0xf0, currBitIndex: 3, buf: []byte{}},                 // 1111 xxxx
			Expected: writerStatus{currByte: 0xf0, currBitIndex: 3, buf: []byte{0xfa, 0xbc, 0xde}}, // 1111 1010 1011 1100 1101 1110 1111 xxxx
		},
		{
			Name:     "pattern 6",
			NBits:    24,
			Value:    0xffabcdef,                                                                   //        10 1010 1111 0011 0111 1011 11
			Start:    writerStatus{currByte: 0xfc, currBitIndex: 1, buf: []byte{}},                 // 1111 11xx
			Expected: writerStatus{currByte: 0xbc, currBitIndex: 1, buf: []byte{0xfe, 0xaf, 0x37}}, // 1111 1110 1010 1111 0011 0111 1011 11xx
		},
		{
			Name:     "pattern 7",
			NBits:    31,
			Value:    0x89abcdef,                                                                   // 0001 0011 0101 0111 1001 1011 1101 111
			Start:    writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{}},                 // xxxx xxxx
			Expected: writerStatus{currByte: 0xde, currBitIndex: 0, buf: []byte{0x13, 0x57, 0x9b}}, // 0001 0011 0101 0111 1001 1011 1101 111x
		},
		{
			Name:     "pattern 8",
			NBits:    31,
			Value:    0x89abcdef,                                                                         //  000 1001 1010 1011 1100 1101 1110 1111
			Start:    writerStatus{currByte: 0x80, currBitIndex: 6, buf: []byte{}},                       // 1xxx xxxx
			Expected: writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{0x89, 0xab, 0xcd, 0xef}}, // 1000 1001 1010 1011 1100 1101 1110 1111
		},
		{
			Name:     "pattern 9",
			NBits:    31,
			Value:    0x89abcdef,                                                                         //   00 0100 1101 0101 1110 0110 1111 0111 1
			Start:    writerStatus{currByte: 0xc0, currBitIndex: 5, buf: []byte{}},                       // 11xx xxxx
			Expected: writerStatus{currByte: 0x80, currBitIndex: 6, buf: []byte{0xc4, 0xd5, 0xe6, 0xf7}}, // 1100 0100 1101 0101 1110 0110 1111 0111 1xxx xxxx
		},
		{
			Name:     "pattern 10",
			NBits:    32,
			Value:    0x89abcdef,                                                                         // 1000 1001 1010 1011 1100 1101 1110 1111
			Start:    writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{}},                       // xxxx xxxx
			Expected: writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{0x89, 0xab, 0xcd, 0xef}}, // 1000 1001 1010 1011 1100 1101 1110 1111 xxxx xxxx
		},
		{
			Name:     "pattern 11",
			NBits:    32,
			Value:    0x89abcdef,                                                                         //  100 0100 1101 0101 1110 0110 1111 0111 1
			Start:    writerStatus{currByte: 0x80, currBitIndex: 6, buf: []byte{}},                       // 1xxx xxxx
			Expected: writerStatus{currByte: 0x80, currBitIndex: 6, buf: []byte{0xc4, 0xd5, 0xe6, 0xf7}}, // 1100 0100 1101 0101 1110 0110 1111 0111 1xxx xxxx
		},
	}

	for _, data := range testData {
		data := data // capture
		t.Run(data.Name, func(t *testing.T) {
			//t.Parallel()

			buf := bytes.NewBuffer(data.Start.buf)
			bw := NewWriter(buf)

			bw.currByte[0] = data.Start.currByte
			bw.currBitIndex = data.Start.currBitIndex

			err := bw.WriteNBitsOfUint32(data.NBits, data.Value)
			if err != nil {
				t.Fatalf("unexpected error: %+v\n", err)
			}
			if uint(data.NBits) != bw.WrittenBits() {
				t.Fatalf("\nunexpected writtenBits\nExpected: %+v\nActual:   %+v\n", data.NBits, bw.WrittenBits())
			}
			if data.Expected.currByte != bw.currByte[0] {
				t.Fatalf("\nunexpected currByte\nExpected: %+v\nActual:   %+v\n", data.Expected.currByte, bw.currByte[0])
			}
			if data.Expected.currBitIndex != bw.currBitIndex {
				t.Fatalf("\nunexpected currBitIndex\nExpected: %+v\nActual:   %+v\n", data.Expected.currBitIndex, bw.currBitIndex)
			}
			if !reflect.DeepEqual(data.Expected.buf, buf.Bytes()) {
				t.Fatalf("\nunexpected flushed data\nExpected: %+v\nActual:   %+v\n", data.Expected.buf, buf.Bytes())
			}

		})
	}

}

func benchmarkWriteNBitsOfUint32(nBits uint8, b *testing.B) {
	rand.Seed(time.Now().UnixNano())
	buf := bytes.NewBuffer([]byte{})
	bw := NewWriter(buf)
	for n := 0; n < b.N; n++ {
		_ = bw.WriteNBitsOfUint32(nBits, uint32(rand.Uint32()))
	}
}

func BenchmarkWrite1BitsOfUint32(b *testing.B) {
	benchmarkWriteNBitsOfUint32(1, b)
}

func BenchmarkWrite16BitsOfUint32(b *testing.B) {
	benchmarkWriteNBitsOfUint32(16, b)
}

func BenchmarkWrite17BitsOfUint32(b *testing.B) {
	benchmarkWriteNBitsOfUint32(17, b)
}

func BenchmarkWrite23BitsOfUint32(b *testing.B) {
	benchmarkWriteNBitsOfUint32(23, b)
}

func BenchmarkWrite31BitsOfUint32(b *testing.B) {
	benchmarkWriteNBitsOfUint32(31, b)
}

func BenchmarkWrite32BitsOfUint32(b *testing.B) {
	benchmarkWriteNBitsOfUint32(32, b)
}

func TestWriteNBits(t *testing.T) {
	testData := []struct {
		Name     string
		NBits    uint8
		Value    []byte
		Start    writerStatus
		Expected writerStatus
	}{
		{
			Name:     "pattern 1",
			NBits:    1,
			Value:    []byte{0xff}, // 1
			Start:    writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{}},
			Expected: writerStatus{currByte: 0x80, currBitIndex: 6, buf: []byte{}},
		},
		{
			Name:     "pattern 2",
			NBits:    16,
			Value:    []byte{0xab, 0xcd},                                                     //        10 1010 1111 0011 01
			Start:    writerStatus{currByte: 0x88, currBitIndex: 1, buf: []byte{}},           // 1000 10xx
			Expected: writerStatus{currByte: 0x34, currBitIndex: 1, buf: []byte{0x8a, 0xaf}}, // 1000 1010 1010 1111 0011 01xx
		},
		{
			Name:     "pattern 3",
			NBits:    17,
			Value:    []byte{0xab, 0xcd, 0xef},                                               //        10 1010 1111 0011 011
			Start:    writerStatus{currByte: 0x88, currBitIndex: 1, buf: []byte{}},           // 1000 10xx
			Expected: writerStatus{currByte: 0x36, currBitIndex: 0, buf: []byte{0x8a, 0xaf}}, // 1000 1010 1010 1111 0011 011x
		},
		{
			Name:     "pattern 4",
			NBits:    24,
			Value:    []byte{0xab, 0xcd, 0xef, 0xff},                                               // 1010 1011 1100 1101 1110 1111
			Start:    writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{}},                 // xxxx xxxx
			Expected: writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{0xab, 0xcd, 0xef}}, // 1010 1011 1100 1101 1110 1111
		},
		{
			Name:     "pattern 5",
			NBits:    24,
			Value:    []byte{0xab, 0xcd, 0xef, 0xff},                                               //      1010 1011 1100 1101 1110 1111
			Start:    writerStatus{currByte: 0xf0, currBitIndex: 3, buf: []byte{}},                 // 1111 xxxx
			Expected: writerStatus{currByte: 0xf0, currBitIndex: 3, buf: []byte{0xfa, 0xbc, 0xde}}, // 1111 1010 1011 1100 1101 1110 1111 xxxx
		},
		{
			Name:     "pattern 6",
			NBits:    24,
			Value:    []byte{0xab, 0xcd, 0xef, 0xff},                                               //        10 1010 1111 0011 0111 1011 11
			Start:    writerStatus{currByte: 0xfc, currBitIndex: 1, buf: []byte{}},                 // 1111 11xx
			Expected: writerStatus{currByte: 0xbc, currBitIndex: 1, buf: []byte{0xfe, 0xaf, 0x37}}, // 1111 1110 1010 1111 0011 0111 1011 11xx
		},
		{
			Name:     "pattern 7",
			NBits:    31,
			Value:    []byte{0x89, 0xab, 0xcd, 0xef},                                               // 1000 1001 1010 1011 1100 1101 1110 111
			Start:    writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{}},                 // xxxx xxxx
			Expected: writerStatus{currByte: 0xee, currBitIndex: 0, buf: []byte{0x89, 0xab, 0xcd}}, // 1000 1001 1010 1011 1100 1101 1110 111x
		},
		{
			Name:     "pattern 8",
			NBits:    31,
			Value:    []byte{0x89, 0xab, 0xcd, 0xef},                                                     //  100 0100 1101 0101 1110 0110 1111 0111
			Start:    writerStatus{currByte: 0x80, currBitIndex: 6, buf: []byte{}},                       // 1xxx xxxx
			Expected: writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{0xc4, 0xd5, 0xe6, 0xf7}}, // 1100 0100 1101 0101 1110 0110 1111 0111 xxxx xxxx
		},
		{
			Name:     "pattern 9",
			NBits:    31,
			Value:    []byte{0x89, 0xab, 0xcd, 0xef},                                                     //   10 0010 0110 1010 1111 0011 0111 1011 1
			Start:    writerStatus{currByte: 0xc0, currBitIndex: 5, buf: []byte{}},                       // 11xx xxxx
			Expected: writerStatus{currByte: 0x80, currBitIndex: 6, buf: []byte{0xe2, 0x6a, 0xf3, 0x7b}}, // 1110 0010 0110 1010 1111 0011 0111 1011 1xxx xxxx
		},
		{
			Name:     "pattern 10",
			NBits:    32,
			Value:    []byte{0x89, 0xab, 0xcd, 0xef},                                                     // 1000 1001 1010 1011 1100 1101 1110 1111
			Start:    writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{}},                       // xxxx xxxx
			Expected: writerStatus{currByte: 0x00, currBitIndex: 7, buf: []byte{0x89, 0xab, 0xcd, 0xef}}, // 1000 1001 1010 1011 1100 1101 1110 1111 xxxx xxxx
		},
		{
			Name:     "pattern 11",
			NBits:    32,
			Value:    []byte{0x89, 0xab, 0xcd, 0xef},                                                     //  100 0100 1101 0101 1110 0110 1111 0111 1
			Start:    writerStatus{currByte: 0x80, currBitIndex: 6, buf: []byte{}},                       // 1xxx xxxx
			Expected: writerStatus{currByte: 0x80, currBitIndex: 6, buf: []byte{0xc4, 0xd5, 0xe6, 0xf7}}, // 1100 0100 1101 0101 1110 0110 1111 0111 1xxx xxxx
		},
	}

	for _, data := range testData {
		data := data // capture
		t.Run(data.Name, func(t *testing.T) {
			//t.Parallel()

			buf := bytes.NewBuffer(data.Start.buf)
			bw := NewWriter(buf)

			bw.currByte[0] = data.Start.currByte
			bw.currBitIndex = data.Start.currBitIndex

			err := bw.WriteNBits(data.NBits, data.Value)
			if err != nil {
				t.Fatalf("unexpected error: %+v\n", err)
			}
			if uint(data.NBits) != bw.WrittenBits() {
				t.Fatalf("\nunexpected writtenBits\nExpected: %+v\nActual:   %+v\n", data.NBits, bw.WrittenBits())
			}
			if data.Expected.currByte != bw.currByte[0] {
				t.Fatalf("\nunexpected currByte\nExpected: %+v\nActual:   %+v\n", data.Expected.currByte, bw.currByte[0])
			}
			if data.Expected.currBitIndex != bw.currBitIndex {
				t.Fatalf("\nunexpected currBitIndex\nExpected: %+v\nActual:   %+v\n", data.Expected.currBitIndex, bw.currBitIndex)
			}
			if !reflect.DeepEqual(data.Expected.buf, buf.Bytes()) {
				t.Fatalf("\nunexpected flushed data\nExpected: %+v\nActual:   %+v\n", data.Expected.buf, buf.Bytes())
			}

		})
	}

}
