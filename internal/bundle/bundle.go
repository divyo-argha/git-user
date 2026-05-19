package bundle

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/scrypt"
)

const version = 1

// Identity holds everything needed to recreate one git-user identity on a new machine.
type Identity struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	PrivateKey []byte `json:"private_key,omitempty"` // raw bytes of the private key file
	PublicKey  []byte `json:"public_key,omitempty"`  // raw bytes of the .pub file
}

type payload struct {
	Version    int        `json:"version"`
	Identities []Identity `json:"identities"`
}

// scrypt params — deliberately expensive
const (
	scryptN = 1 << 17 // 128 MB memory
	scryptR = 8
	scryptP = 1
	keyLen  = 32
	saltLen = 32
)

// Encrypt serialises identities and encrypts them with passphrase using
// AES-256-GCM. The returned bytes are: salt(32) + nonce(12) + ciphertext.
func Encrypt(identities []Identity, passphrase string) ([]byte, error) {
	plain, err := json.Marshal(payload{Version: version, Identities: identities})
	if err != nil {
		return nil, fmt.Errorf("encoding bundle: %w", err)
	}

	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}

	key, err := scrypt.Key([]byte(passphrase), salt, scryptN, scryptR, scryptP, keyLen)
	if err != nil {
		return nil, fmt.Errorf("deriving key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plain, nil)

	out := make([]byte, 0, saltLen+len(nonce)+len(ciphertext))
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

// Decrypt reverses Encrypt. Returns the identities on success.
func Decrypt(data []byte, passphrase string) ([]Identity, error) {
	if len(data) < saltLen+12+1 {
		return nil, errors.New("bundle too short or corrupt")
	}

	salt := data[:saltLen]
	rest := data[saltLen:]

	key, err := scrypt.Key([]byte(passphrase), salt, scryptN, scryptR, scryptP, keyLen)
	if err != nil {
		return nil, fmt.Errorf("deriving key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(rest) < gcm.NonceSize() {
		return nil, errors.New("bundle too short or corrupt")
	}
	nonce := rest[:gcm.NonceSize()]
	ciphertext := rest[gcm.NonceSize():]

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed — wrong passphrase or corrupt file")
	}

	var p payload
	if err := json.Unmarshal(plain, &p); err != nil {
		return nil, fmt.Errorf("decoding bundle: %w", err)
	}
	return p.Identities, nil
}
