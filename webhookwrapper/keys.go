package webhookwrapper

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

type keyPair struct {
	PublicKey  string
	PrivateKey string
	Key        *rsa.PrivateKey
}

func newPrivateKey(keyBitSize int) (*keyPair, error) {
	key, err := rsa.GenerateKey(rand.Reader, keyBitSize)
	if err != nil {
		return nil, err
	}
	publicKey, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, err
	}
	publicKeyPEM := new(bytes.Buffer)
	err = pem.Encode(publicKeyPEM, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKey,
	})
	if err != nil {
		return nil, err
	}
	privateKeyPEM := new(bytes.Buffer)
	err = pem.Encode(privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err != nil {
		return nil, err
	}
	return &keyPair{
		PublicKey:  publicKeyPEM.String(),
		PrivateKey: privateKeyPEM.String(),
		Key:        key,
	}, nil
}
