package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func TestParseSSHKeyFingerprint(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"2048 SHA256:abcde12345/xyz user@host (RSA)", "SHA256:abcde12345/xyz", false},
		{"   3072   SHA256:abcdef   ", "SHA256:abcdef", false},
		{"invalid_line", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		got, err := ParseSSHKeyFingerprint(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseSSHKeyFingerprint(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseSSHKeyFingerprint(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestVerifyPassphrase(t *testing.T) {
	// Create temporary directory for mock keys
	tmpDir, err := os.MkdirTemp("", "ssh-test-*")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Generate an unencrypted RSA private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}

	unencryptedPath := filepath.Join(tmpDir, "unencrypted")
	unencryptedFile, err := os.Create(unencryptedPath)
	if err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	pem.Encode(unencryptedFile, pemBlock)
	unencryptedFile.Close()

	// Verify unencrypted key with no passphrase
	if !VerifyPassphrase(unencryptedPath, "") {
		t.Errorf("Expected true for unencrypted key with empty passphrase")
	}

	// 2. Generate an encrypted RSA private key
	passphrase := "test-secret"
	// Marshal encrypted key using x509
	//lint:ignore SA1019 Deliberate: produce legacy encrypted PEM to verify backward-compatible parsing.
	encBlock, err := x509.EncryptPEMBlock(rand.Reader, "RSA PRIVATE KEY", privBytes, []byte(passphrase), x509.PEMCipherAES256)
	if err != nil {
		t.Fatalf("Failed to encrypt PEM block: %v", err)
	}

	encryptedPath := filepath.Join(tmpDir, "encrypted")
	encryptedFile, err := os.Create(encryptedPath)
	if err != nil {
		t.Fatalf("Failed to create encrypted key file: %v", err)
	}
	pem.Encode(encryptedFile, encBlock)
	encryptedFile.Close()

	// Verify with correct passphrase
	if !VerifyPassphrase(encryptedPath, passphrase) {
		t.Errorf("Expected true for correct passphrase")
	}

	// Verify with incorrect passphrase
	if VerifyPassphrase(encryptedPath, "wrong-passphrase") {
		t.Errorf("Expected false for incorrect passphrase")
	}

	// Verify with missing file
	if VerifyPassphrase(filepath.Join(tmpDir, "does-not-exist"), passphrase) {
		t.Errorf("Expected false for non-existent file")
	}
}

func TestEnsureSSHAgent(t *testing.T) {
	// Temporarily clean/mock env
	originalAuthSock := os.Getenv("SSH_AUTH_SOCK")
	defer os.Setenv("SSH_AUTH_SOCK", originalAuthSock)

	os.Setenv("SSH_AUTH_SOCK", "/tmp/mock-ssh-sock")
	if err := EnsureSSHAgent(); err != nil {
		t.Errorf("Expected nil error when SSH_AUTH_SOCK is set, got %v", err)
	}
}

func TestWithMockedAgent(t *testing.T) {
	// Setup a mocked SSH Agent running on a local temporary Unix socket
	tmpDir, err := os.MkdirTemp("", "ssh-agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sockPath := filepath.Join(tmpDir, "agent.sock")
	l, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("Failed to listen on unix socket: %v", err)
	}
	defer l.Close()

	// In-memory keyring agent
	memAgent := agent.NewKeyring()

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go agent.ServeAgent(memAgent, conn)
		}
	}()

	// Set environment variable to route GetAgentClient to our server
	originalAuthSock := os.Getenv("SSH_AUTH_SOCK")
	os.Setenv("SSH_AUTH_SOCK", sockPath)
	defer os.Setenv("SSH_AUTH_SOCK", originalAuthSock)

	// Create test RSA key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}

	keyPath := filepath.Join(tmpDir, "id_rsa")
	keyFile, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("Failed to create private key: %v", err)
	}
	pem.Encode(keyFile, pemBlock)
	keyFile.Close()

	// Create public key companion file
	pubKey, err := ssh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("Failed to create ssh public key: %v", err)
	}
	pubBytes := ssh.MarshalAuthorizedKey(pubKey)
	err = os.WriteFile(keyPath+".pub", pubBytes, 0644)
	if err != nil {
		t.Fatalf("Failed to write public key file: %v", err)
	}

	// Test IsSSHKeyLoaded before loading
	if IsSSHKeyLoaded(keyPath) {
		t.Errorf("Expected key not loaded initially")
	}

	// Test AddSSHKeyWithPassphrase
	err = AddSSHKeyWithPassphrase(keyPath, "")
	if err != nil {
		t.Fatalf("AddSSHKeyWithPassphrase returned error: %v", err)
	}

	// Test IsSSHKeyLoaded after loading
	if !IsSSHKeyLoaded(keyPath) {
		t.Errorf("Expected key to be loaded after AddSSHKeyWithPassphrase")
	}

	// Test LoadedSSHKeyFingerprints
	fingerprints, err := LoadedSSHKeyFingerprints()
	if err != nil {
		t.Errorf("Failed to list loaded fingerprints: %v", err)
	}
	if len(fingerprints) != 1 {
		t.Errorf("Expected 1 loaded fingerprint, got %d", len(fingerprints))
	}

	// Test RemoveSSHKey
	err = RemoveSSHKey(keyPath)
	if err != nil {
		t.Errorf("RemoveSSHKey failed: %v", err)
	}

	// Test IsSSHKeyLoaded after removal
	if IsSSHKeyLoaded(keyPath) {
		t.Errorf("Expected key to be unloaded after RemoveSSHKey")
	}
}

func TestEnsureSSHAgentError(t *testing.T) {
	// Temporarily clean/mock env to be empty
	originalAuthSock := os.Getenv("SSH_AUTH_SOCK")
	os.Setenv("SSH_AUTH_SOCK", "")
	defer os.Setenv("SSH_AUTH_SOCK", originalAuthSock)

	// Since we are running on mac (GOOS=darwin), it should warn and return an error
	err := EnsureSSHAgent()
	if err == nil {
		t.Error("Expected error when SSH_AUTH_SOCK is empty")
	}
}

func TestGetAgentClientError(t *testing.T) {
	originalAuthSock := os.Getenv("SSH_AUTH_SOCK")
	os.Setenv("SSH_AUTH_SOCK", "")
	defer os.Setenv("SSH_AUTH_SOCK", originalAuthSock)

	_, _, err := GetAgentClient()
	if err == nil || !strings.Contains(err.Error(), "SSH_AUTH_SOCK not set") {
		t.Errorf("Expected 'SSH_AUTH_SOCK not set' error, got %v", err)
	}

	// Invalid socket path should cause dial failure
	os.Setenv("SSH_AUTH_SOCK", "/nonexistent/path/socket.sock")
	_, _, err = GetAgentClient()
	if err == nil {
		t.Error("Expected error for invalid socket dial")
	}
}

func TestSSHKeyFingerprintErrors(t *testing.T) {
	_, err := SSHKeyFingerprint("nonexistent-key")
	if err == nil {
		t.Error("Expected error for nonexistent key")
	}

	// Create a temp file with invalid public key data
	tmpDir, err := os.MkdirTemp("", "ssh-key-test-*")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	invalidKey := filepath.Join(tmpDir, "invalid_key")
	err = os.WriteFile(invalidKey, []byte("not-pem-data"), 0600)
	if err != nil {
		t.Fatalf("Failed to write invalid key: %v", err)
	}

	err = os.WriteFile(invalidKey+".pub", []byte("not-authorized-key"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid pub key: %v", err)
	}

	// This should fail to parse, and fall back to ssh-keygen which will fail, returning an error
	_, err = SSHKeyFingerprint(invalidKey)
	if err == nil {
		t.Error("Expected error for invalid key files")
	}
}

func TestEnsureMacOSKeychainConfigured(t *testing.T) {
	if runtime.GOOS != "darwin" {
		err := EnsureMacOSKeychainConfigured()
		if err != nil {
			t.Errorf("EnsureMacOSKeychainConfigured failed on non-darwin: %v", err)
		}
		return
	}

	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	tmpDir, err := os.MkdirTemp("", "ssh-home-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp home dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("HOME", tmpDir)

	err = EnsureMacOSKeychainConfigured()
	if err != nil {
		t.Fatalf("EnsureMacOSKeychainConfigured failed: %v", err)
	}

	configPath := filepath.Join(tmpDir, ".ssh", "config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read created ssh config: %v", err)
	}

	if !strings.Contains(string(data), "UseKeychain yes") {
		t.Errorf("Expected config to contain UseKeychain yes, got %s", string(data))
	}

	// Run again to ensure idempotency (should not duplicate or error)
	err = EnsureMacOSKeychainConfigured()
	if err != nil {
		t.Errorf("Subsequent run failed: %v", err)
	}
}

func TestAddKeyToMacOSKeychain(t *testing.T) {
	// Call it with an invalid file, it should fall through and return nil
	err := addKeyToMacOSKeychain("nonexistent-key", "some-pass")
	if err != nil {
		t.Errorf("addKeyToMacOSKeychain failed: %v", err)
	}
}
