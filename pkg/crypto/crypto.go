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

// GenerateKeys creates a new 2048-bit RSA public/private key pair in PEM format.
func GenerateKeys() (privateKeyPEM, publicKeyPEM []byte, err error) {
	// 1. Generate an RSA private key.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("could not generate rsa key: %w", err)
	}

	// 2. Encode the private key into PKCS1 PEM format.
	privateKeyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// 3. Derive the public key and encode it into PKIX PEM format.
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("could not marshal public key: %w", err)
	}
	publicKeyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return privateKeyPEM, publicKeyPEM, nil
}

// Encrypt performs hybrid encryption (AES-GCM + RSA).
// It encrypts the data with a new AES key, then encrypts the AES key with the RSA public key.
// The provided 'aad' is authenticated but not encrypted.
func Encrypt(data, aad, publicKeyPEM []byte) (encryptedKey, encryptedData []byte, err error) {
	// 1. Generate a new random AES key for this message only.
	aesKey := make([]byte, 32) // AES-256
	if _, err := rand.Read(aesKey); err != nil {
		return nil, nil, fmt.Errorf("could not generate AES key: %w", err)
	}

	// 2. Encrypt the data with the AES key using GCM mode.
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
	// The nonce is prepended to the ciphertext. The AAD is authenticated.
	encryptedData = gcm.Seal(nonce, nonce, data, aad)

	// 3. Encrypt the AES key with the recipient's RSA public key.
	pubBlock, _ := pem.Decode(publicKeyPEM)
	if pubBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode PEM block containing public key")
	}
	pub, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse public key: %w", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, nil, fmt.Errorf("key is not an RSA public key")
	}

	encryptedKey, err = rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPub, aesKey, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("could not encrypt AES key: %w", err)
	}

	return encryptedKey, encryptedData, nil
}

// Decrypt performs hybrid decryption.
// It decrypts the AES key with the RSA private key, then decrypts the data with the AES key.
// The provided 'aad' must match the one used during encryption, or this function will fail.
func Decrypt(encryptedKey, encryptedData, aad, privateKeyPEM []byte) ([]byte, error) {
	// 1. Decrypt the AES key with our RSA private key.
	privBlock, _ := pem.Decode(privateKeyPEM)
	if privBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing private key")
	}
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

	// Decrypt and VERIFY. If the AAD is incorrect, this will produce an authentication error.
	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt or verify data (tampering detected): %w", err)
	}

	return plaintext, nil
}

// Sign creates a digital signature for a message using a private key.
func Sign(msg, privateKeyPEM []byte) ([]byte, error) {
	// 1. Hash the message that we are signing.
	hashed := sha256.Sum256(msg)

	// 2. Parse the private key.
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing private key")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse private key for signing: %w", err)
	}

	// 3. Sign the hash.
	return rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hashed[:])
}

// Verify checks a signature against a message and a public key.
func Verify(msg, sig, publicKeyPEM []byte) error {
	// 1. Hash the message to get the digest that was signed.
	hashed := sha256.Sum256(msg)

	// 2. Parse the public key.
	block, _ := pem.Decode(publicKeyPEM)
	if block == nil {
		return fmt.Errorf("failed to decode PEM block containing public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("could not parse public key for verification: %w", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("key is not an RSA public key")
	}

	// 3. Verify the signature against the hash.
	return rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, hashed[:], sig)
}
