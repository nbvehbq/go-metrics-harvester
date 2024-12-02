package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"hash"
	"io"
	"net/http"
)

func WithDecrypt(key []byte) func(http.HandlerFunc) http.HandlerFunc {
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			originalWriter := w
			if key != nil {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "can't read body", http.StatusBadRequest)
					return
				}

				privateKeyBlock, _ := pem.Decode(key)
				privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				plaintBody, err := DecryptOAEP(sha256.New(), privateKey, body, nil)
				if err != nil {
					http.Error(w, err.Error(), http.StatusForbidden)
					return
				}

				r.Body = io.NopCloser(bytes.NewBuffer(plaintBody))
			}

			h.ServeHTTP(originalWriter, r)
		}
	}
}

func EncryptOAEP(hash hash.Hash, public *rsa.PublicKey, msg []byte, label []byte) ([]byte, error) {
	msgLen := len(msg)
	step := public.Size() - 2*hash.Size() - 2
	var encryptedBytes []byte

	for start := 0; start < msgLen; start += step {
		finish := start + step
		if finish > msgLen {
			finish = msgLen
		}

		encryptedBlockBytes, err := rsa.EncryptOAEP(hash, rand.Reader, public, msg[start:finish], label)
		if err != nil {
			return nil, err
		}

		encryptedBytes = append(encryptedBytes, encryptedBlockBytes...)
	}

	return encryptedBytes, nil
}

func DecryptOAEP(hash hash.Hash, private *rsa.PrivateKey, msg []byte, label []byte) ([]byte, error) {
	msgLen := len(msg)
	step := private.PublicKey.Size()
	var decryptedBytes []byte

	for start := 0; start < msgLen; start += step {
		finish := start + step
		if finish > msgLen {
			finish = msgLen
		}

		decryptedBlockBytes, err := rsa.DecryptOAEP(hash, nil, private, msg[start:finish], label)
		if err != nil {
			return nil, err
		}

		decryptedBytes = append(decryptedBytes, decryptedBlockBytes...)
	}

	return decryptedBytes, nil
}

func GenerateCert() ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	certBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	var certPEM bytes.Buffer
	pem.Encode(&certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	var privateKeyPEM bytes.Buffer
	pem.Encode(&privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return certPEM.Bytes(), privateKeyPEM.Bytes(), nil
}
