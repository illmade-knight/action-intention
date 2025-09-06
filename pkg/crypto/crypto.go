// FILE: pkg/crypto/crypto.go

package crypto

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// GenerateKeys creates a new RSA public/private key pair in PEM format.
func GenerateKeys() (privateKeyPEM, publicKeyPEM []byte, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	privateKeyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	publicKeyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return privateKeyPEM, publicKeyPEM, nil
}

// Sign creates a signature for a message using a private key.
func Sign(msg, privateKeyPEM []byte) ([]byte, error) {
	block, _ := pem.Decode(privateKeyPEM)
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	hashed := sha256.Sum256(msg)
	return rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hashed[:])
}

// Verify checks a signature against a message and public key.
func Verify(msg, sig, publicKeyPEM []byte) error {
	block, _ := pem.Decode(publicKeyPEM)
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}
	rsaPub, _ := pub.(*rsa.PublicKey)
	hashed := sha256.Sum256(msg)
	return rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, hashed[:], sig)
}

// Encrypt performs hybrid encryption (AES + RSA).
// It returns the RSA-encrypted AES key and the AES-encrypted data.
func Encrypt(data, publicKeyPEM []byte) (encryptedKey, encryptedData []byte, err error) {
	// 1. Generate a new random AES key for this message only.
	aesKey := make([]byte, 32) // AES-256
	if _, err := rand.Read(aesKey); err != nil {
		return nil, nil, fmt.Errorf("could not generate AES key: %w", err)
	}

	// 2. Encrypt the data with the AES key.
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("could not create nonce: %w", err)
	}
	encryptedData = gcm.Seal(nonce, nonce, data, nil)

	// 3. Encrypt the AES key with the recipient's RSA public key.
	pubBlock, _ := pem.Decode(publicKeyPEM)
	pub, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse public key: %w", err)
	}
	rsaPub, _ := pub.(*rsa.PublicKey)
	encryptedKey, err = rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPub, aesKey, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("could not encrypt AES key: %w", err)
	}

	return encryptedKey, encryptedData, nil
}

// Decrypt performs hybrid decryption.
func Decrypt(encryptedKey, encryptedData, privateKeyPEM []byte) ([]byte, error) {
	// 1. Decrypt the AES key with our RSA private key.
	privBlock, _ := pem.Decode(privateKeyPEM)
	priv, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse private key: %w", err)
	}
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encryptedKey, nil)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt AES key: %w", err)
	}

	// 2. Decrypt the data with the recovered AES key.
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("could not create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("could not create GCM: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt data: %w", err)
	}

	return plaintext, nil
}
