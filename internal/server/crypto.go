package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"os"
)

type Decryptor struct {
	privateKey *rsa.PrivateKey
	decrypted  []byte
}

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
	var err error
	var label []byte
	hash := sha256.New()
	d.decrypted, err = rsa.DecryptOAEP(hash, rand.Reader, d.privateKey, msg, label)
	if err != nil {
		return nil, err
	}

	return d.decrypted, nil
}
