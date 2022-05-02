package tools

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
)

func RsaEncry(PublicKey string, content string) string {
	key, _ := base64.StdEncoding.DecodeString(PublicKey)
	pubKey, _ := x509.ParsePKIXPublicKey(key)
	encryptedData, _ := rsa.EncryptPKCS1v15(rand.Reader, pubKey.(*rsa.PublicKey), []byte(content))
	ss := base64.StdEncoding.EncodeToString(encryptedData)
	return ss
}
