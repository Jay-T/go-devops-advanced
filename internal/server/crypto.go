package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"os"
)

// Decryptor struct contains decryption key.
type Decryptor struct {
	privateKey *rsa.PrivateKey
	decrypted  []byte
}

// NewDecryptor returns a Decryptor.
func NewDecryptor(privateKeyFile string) (*Decryptor, error) {
	file, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return nil, err
	}
	d := &Decryptor{}

	block, _ := pem.Decode(file)
	d.privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)

	if err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Decryptor) decrypt(msg []byte) ([]byte, error) {
	var label []byte
	hash := sha256.New()

	msgLen := len(msg)
	step := d.privateKey.Size()
	var decryptedBytes []byte

	for start := 0; start < msgLen; start += step {
		finish := start + step
		if finish > msgLen {
			finish = msgLen
		}

		decryptedBlockBytes, err := rsa.DecryptOAEP(hash, rand.Reader, d.privateKey, msg[start:finish], label)
		if err != nil {
			return nil, err
		}

		decryptedBytes = append(decryptedBytes, decryptedBlockBytes...)
	}

	return decryptedBytes, nil
}
