package bitstream

import (
	"bytes"
	"reflect"
	"testing"
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
}

func BenchmarkWriteBit(b *testing.B) {

}

func TestWriteNBitsOfUint8(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})
	bw := NewWriter(buf)

	bw.WriteNBitsOfUint8(1, 0xff) // writes LBS 1 bit of 0xff == 1b
	if bw.currByte[0] != 0x80 || bw.currBitIndex != 6 {
		t.Fatalf("")
	}
	bw.WriteNBitsOfUint8(2, 0x55) // writes LSB 2 bits of 0x55 = 01b, so far 101b
	if bw.currByte[0] != 0xa0 || bw.currBitIndex != 4 {
		t.Fatalf("")
	}
	bw.WriteNBitsOfUint8(3, 0xf5) // writes LSB 3 bits of 0xf5 = 101b, so far 101101b
	if bw.currByte[0] != 0xb4 || bw.currBitIndex != 1 {
		t.Fatalf(bw.dump())
	}
	bw.WriteNBitsOfUint8(4, 0xfa) // writes LSB 4 bits of 0x0a = 1010b, so far 10110110 10b
	if len(buf.Bytes()) != 1 || buf.Bytes()[0] != 0xb6 || bw.currByte[0] != 0x80 || bw.currBitIndex != 5 {
		t.Fatalf(bw.dump())
	}
}
