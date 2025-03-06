package codec

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func TestNewLengthCodec(t *testing.T) {
	c := NewLengthCodec(0xABAB, 1024)
	d, _ := c.Marshal([]byte("hello"))
	fmt.Printf("%v\n", d)
}

func TestLengthCodec_Marshal(t *testing.T) {
	magicNumberBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(magicNumberBytes, 0xAB)
	println(magicNumberBytes)
}
