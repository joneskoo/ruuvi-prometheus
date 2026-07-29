package hci

import (
	"bytes"
	"testing"
)

// Test vector for Random Address Hash function ah,
// taken from Bluetooth 5.0, vol 3, part H, Appendix D, ch D.7
func TestAh(t *testing.T) {
	irk := []byte{0xec, 0x02, 0x34, 0xa3, 0x57, 0xc8, 0xad, 0x05,
		0x34, 0x10, 0x10, 0xa6, 0x0a, 0x39, 0x7d, 0x9b}
	prand := []byte{0x94, 0x81, 0x70}

	hash := ah(irk, prand)
	if hash == nil {
		t.Fatal()
	}
	if len(hash) != 3 {
		t.Errorf("ah return value should be 3 bytes, was %d", len(hash))
	}

	expected := []byte{0xaa, 0xfb, 0x0d}
	if !bytes.Equal(hash, expected) {
		t.Errorf("Hash does not match (hash %v, expected %v", hash, expected)
	}
}
