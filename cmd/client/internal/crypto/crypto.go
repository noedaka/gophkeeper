package crypto

import (
	"crypto/rand"
	"errors"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/nacl/secretbox"
)

type Crypt struct {
	Key [32]byte
}

const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64 MiB
	argonThreads = 4
)

func NewCrypt(password string, username string) *Crypt {
	salt := []byte(username)

	keyBytes := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, 32)

	var key [32]byte
	copy(key[:], keyBytes)

	return &Crypt{Key: key}
}

func (c *Crypt) Encrypt(plain []byte) ([]byte, []byte, error) {
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, nil, err
	}
	encrypted := secretbox.Seal(nil, plain, &nonce, &c.Key)
	return encrypted, nonce[:], nil
}

func (c *Crypt) Decrypt(encrypted, nonce []byte) ([]byte, error) {
	if len(nonce) != 24 {
		return nil, errors.New("invalid nonce")
	}
	var n [24]byte
	copy(n[:], nonce)
	plain, ok := secretbox.Open(nil, encrypted, &n, &c.Key)
	if !ok {
		return nil, errors.New("decryption failed")
	}
	return plain, nil
}
