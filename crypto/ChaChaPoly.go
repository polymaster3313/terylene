package zcrypto

import (
	"crypto/rand"
	"errors"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

func EncryptChaCha20Poly1305(plaintext []byte, key []byte) ([]byte, error) {
	block, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, block.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := block.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

func DecryptChaCha20Poly1305(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < block.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce := ciphertext[:block.NonceSize()]
	ciphertext = ciphertext[block.NonceSize():]

	plaintext, err := block.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
