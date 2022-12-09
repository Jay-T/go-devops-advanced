package agent

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

type keyError struct {
	s string
}

func (e *keyError) Error() string {
	return e.s
}

func NewKeyError(text string) error {
	return &keyError{text}
}

type Encryptor struct {
	publicKey *rsa.PublicKey
	encrypted []byte
}

func NewEncryptor(publicKeyFile string) (*Encryptor, error) {
	e := &Encryptor{}

	file, err := os.ReadFile(publicKeyFile)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(file)

	pkey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := pkey.(*rsa.PublicKey)
	if !ok {
		return nil, NewKeyError(fmt.Sprintf("got unexpected key type: %T", pkey))
	}
	e.publicKey = rsaKey

	return e, nil
}

func (e *Encryptor) encrypt(msg []byte) ([]byte, error) {
	var err error
	var label []byte
	hash := sha256.New()
	if e.encrypted, err = rsa.EncryptOAEP(hash, rand.Reader, e.publicKey, msg, label); err != nil {
		return nil, err
	}
	return e.encrypted, nil
}
