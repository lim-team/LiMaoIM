package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// RSADecrypt RSA 解码
func RSADecrypt(ciphertext string, privateKey string) ([]byte, error) {
	var privateKeyBytes = []byte(`
	-----BEGIN RSA PRIVATE KEY-----
	` + privateKey + `
	-----END RSA PRIVATE KEY-----
	`)
	block, _ := pem.Decode(privateKeyBytes)
	if block == nil {
		return nil, errors.New("Private key error！")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.DecryptPKCS1v15(rand.Reader, priv, []byte(ciphertext))
}

// RSAEncrypt RSA编码
func RSAEncrypt(data string, publicKey string) ([]byte, error) {
	var publicKeyBytes = []byte(`
	-----BEGIN PUBLIC KEY-----
	` + publicKey + `
	-----END PUBLIC KEY-----
	`)
	block, _ := pem.Decode(publicKeyBytes)
	if block == nil {
		return nil, errors.New("public key error")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub := pubInterface.(*rsa.PublicKey)
	return rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(data))
}
