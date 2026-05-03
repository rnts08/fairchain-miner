package algorithm

import (
	"crypto/sha256"
	"encoding/binary"
	"reflect"
	"testing"
)

func TestSHANIInternal(t *testing.T) {
	if !HasSHANI {
		t.Skip("SHA-NI not supported on this CPU")
	}

	// Initial SHA-256 state
	state := [8]uint32{
		0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
		0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
	}

	// Block for empty message: 0x80 + 55 zeros + 8 bytes of 0
	var block [64]byte
	block[0] = 0x80
	binary.BigEndian.PutUint64(block[56:], 0)

	want := sha256.Sum256([]byte{})

	sha256_compress_single_shani(&state, &block)

	var got [32]byte
	for i := 0; i < 8; i++ {
		binary.BigEndian.PutUint32(got[i*4:], state[i])
	}

	if !reflect.DeepEqual(got[:], want[:]) {
		t.Errorf("Internal hash mismatch\ngot:  %x\nwant: %x", got, want)
	}
}
