package crypto

import (
	"crypto/rand"
	"errors"

	"golang.org/x/crypto/nacl/secretbox"
)

var TestKey = [32]byte{
	0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
	0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
	0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
	0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
}

func Encrypt(plain []byte) ([]byte, []byte, error) {
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, nil, err
	}
	encrypted := secretbox.Seal(nil, plain, &nonce, &TestKey)
	return encrypted, nonce[:], nil
}

func Decrypt(encrypted, nonce []byte) ([]byte, error) {
	if len(nonce) != 24 {
		return nil, errors.New("invalid nonce")
	}
	var n [24]byte
	copy(n[:], nonce)
	plain, ok := secretbox.Open(nil, encrypted, &n, &TestKey)
	if !ok {
		return nil, errors.New("decryption failed")
	}
	return plain, nil
}
