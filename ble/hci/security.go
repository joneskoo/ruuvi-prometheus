package hci

import (
	"crypto/aes"
)

const (
	//IrkLength defines the required length for IRK
	IrkLength int = 16
	//PrandLength defines the required length for PRAND
	PrandLength int = 3
)

// Random Address Hash function
// See Bluetooth 5.0, vol 3, part H, ch 2.2.2
func ah(k []byte, r []byte) []byte {

	if len(k) != IrkLength || len(r) != PrandLength {
		return nil
	}

	// r is concatenated with padding to generate r’ which is used as
	// the 128-bit input parameter plaintextData to security function e:
	// r’ = padding || r
	// The least significant octet of r becomes the least significant octet
	// of r’ and the most significant octet of padding becomes the most
	// significant octet of r

	// NOTE: for AES, the input format is MSB in position 0, Big Endian
	// We try to use the canonical Bluetooth byte ordering, that is,
	// little endian, for storing addresses and such, and thus we need to swap
	// here.

	rpadded := make([]byte, aes.BlockSize)
	rpadded[15] = r[0]
	rpadded[14] = r[1]
	rpadded[13] = r[2]

	// The output of the random address function ah is:
	// ah(k, r) = e(k, r’) mod 2^24
	// The output of the security function e is then truncated to 24 bits by
	// taking the least significant 24 bits of the output of e as the result
	// of ah.
	cipher, err := aes.NewCipher(k)
	if err != nil {
		return nil
	}
	ret := make([]byte, aes.BlockSize)
	cipher.Encrypt(ret, rpadded)
	rvalue := make([]byte, 3)
	// return the value in little-endian, that is, LSB on position 0
	rvalue[0] = ret[15]
	rvalue[1] = ret[14]
	rvalue[2] = ret[13]
	return rvalue
}
