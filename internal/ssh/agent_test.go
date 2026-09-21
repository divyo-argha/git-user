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
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// startTestAgent serves backing over a temp unix socket, points SSH_AUTH_SOCK
// at it for the duration of the test, and returns a freshly generated RSA
// key pair's private key path. Mirrors the setup in TestWithMockedAgent
// (ssh_test.go), kept self-contained here since it also needs to wrap the
// backing agent.Agent for TestAddSSHKeyWithOptions_ConfirmBeforeUse.
func startTestAgent(t *testing.T, backing agent.Agent) string {
	t.Helper()
	tmpDir := t.TempDir()

	sockPath := filepath.Join(tmpDir, "agent.sock")
	l, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listening on unix socket: %v", err)
	}
	t.Cleanup(func() { l.Close() })

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go agent.ServeAgent(backing, conn)
		}
	}()

	t.Setenv("SSH_AUTH_SOCK", sockPath)

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating test key: %v", err)
	}
	pemBlock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}

	keyPath := filepath.Join(tmpDir, "id_rsa")
	keyFile, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("creating test key file: %v", err)
	}
	if err := pem.Encode(keyFile, pemBlock); err != nil {
		t.Fatalf("encoding test key: %v", err)
	}
	keyFile.Close()

	pubKey, err := ssh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("deriving test public key: %v", err)
	}
	if err := os.WriteFile(keyPath+".pub", ssh.MarshalAuthorizedKey(pubKey), 0644); err != nil {
		t.Fatalf("writing test public key: %v", err)
	}

	return keyPath
}

// TestAddSSHKeyWithOptions_Lifetime confirms LifetimeSecs actually bounds how
// long a key stays loaded: the stock in-memory agent.NewKeyring() honors it
// (it's what backs agent.ServeAgent's real expiry enforcement), so this is a
// genuine end-to-end check of the option reaching the agent, not just of the
// struct literal being built correctly.
func TestAddSSHKeyWithOptions_Lifetime(t *testing.T) {
	keyPath := startTestAgent(t, agent.NewKeyring())

	if err := AddSSHKeyWithOptions(keyPath, "", AgentLoadOptions{LifetimeSecs: 1}); err != nil {
		t.Fatalf("AddSSHKeyWithOptions: %v", err)
	}
	if !IsSSHKeyLoaded(keyPath) {
		t.Fatalf("expected key to be loaded immediately after AddSSHKeyWithOptions")
	}

	time.Sleep(1200 * time.Millisecond)

	if IsSSHKeyLoaded(keyPath) {
		t.Errorf("expected key to have expired from the agent after its 1s lifetime")
	}
}

// fakeAgent wraps a real agent.Agent and records the last AddedKey passed to
// Add() — used because the stock agent.NewKeyring() mock silently drops
// ConfirmBeforeUse (it only ever reads LifetimeSecs), so it can't itself
// prove the constraint was set. Serving this behind the same real
// agent.ServeAgent socket harness as the other tests still exercises the
// real wire-protocol encode/decode of ConfirmBeforeUse.
type fakeAgent struct {
	agent.Agent
	lastAdded agent.AddedKey
}

func (f *fakeAgent) Add(key agent.AddedKey) error {
	f.lastAdded = key
	return f.Agent.Add(key)
}

func TestAddSSHKeyWithOptions_ConfirmBeforeUse(t *testing.T) {
	fake := &fakeAgent{Agent: agent.NewKeyring()}
	keyPath := startTestAgent(t, fake)

	if err := AddSSHKeyWithOptions(keyPath, "", AgentLoadOptions{ConfirmBeforeUse: true}); err != nil {
		t.Fatalf("AddSSHKeyWithOptions: %v", err)
	}
	if !fake.lastAdded.ConfirmBeforeUse {
		t.Errorf("expected ConfirmBeforeUse to reach the agent's Add() call, got AddedKey=%+v", fake.lastAdded)
	}
}

func TestSSHAddArgs(t *testing.T) {
	cases := []struct {
		name             string
		opts             AgentLoadOptions
		useAppleKeychain bool
		want             []string
	}{
		{"no options", AgentLoadOptions{}, false, []string{"/key"}},
		{"lifetime only", AgentLoadOptions{LifetimeSecs: 3600}, false, []string{"-t", "3600", "/key"}},
		{"confirm only", AgentLoadOptions{ConfirmBeforeUse: true}, false, []string{"-c", "/key"}},
		{"lifetime and confirm", AgentLoadOptions{LifetimeSecs: 60, ConfirmBeforeUse: true}, false, []string{"-t", "60", "-c", "/key"}},
		// useAppleKeychain=true is deliberately not covered here — whether
		// --apple-use-keychain appears also depends on runtime.GOOS, which
		// this table can't vary; see TestSSHAddArgsAppleKeychain.
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sshAddArgs("/key", c.opts, c.useAppleKeychain)
			if len(got) != len(c.want) {
				t.Fatalf("sshAddArgs(%+v) = %v, want %v", c.opts, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("sshAddArgs(%+v) = %v, want %v", c.opts, got, c.want)
				}
			}
		})
	}
}

// TestSSHAddArgsAppleKeychain checks the platform-conditional flag
// separately: --apple-use-keychain only appears when both useAppleKeychain
// is true AND the build is darwin, so its presence/absence depends on
// runtime.GOOS rather than being a fixed expectation like the other cases.
func TestSSHAddArgsAppleKeychain(t *testing.T) {
	args := sshAddArgs("/key", AgentLoadOptions{}, true)
	hasFlag := false
	for _, a := range args {
		if a == "--apple-use-keychain" {
			hasFlag = true
		}
	}
	wantFlag := runtime.GOOS == "darwin"
	if hasFlag != wantFlag {
		t.Errorf("sshAddArgs with useAppleKeychain=true: --apple-use-keychain present=%v, want %v (GOOS-dependent)", hasFlag, wantFlag)
	}
}
