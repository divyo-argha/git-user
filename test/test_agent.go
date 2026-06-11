//go:build ignore

package main

import (
	"fmt"
	"net"
	"os"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: go run test_agent.go <key_path> <passphrase>")
		return
	}
	keyPath := os.Args[1]
	passphrase := os.Args[2]

	data, err := os.ReadFile(keyPath)
	if err != nil {
		fmt.Printf("read file err: %v\n", err)
		return
	}

	privKey, err := ssh.ParseRawPrivateKeyWithPassphrase(data, []byte(passphrase))
	if err != nil {
		fmt.Printf("ParseRawPrivateKeyWithPassphrase err: %v\n", err)
		return
	}

	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		fmt.Println("SSH_AUTH_SOCK is empty")
		return
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		fmt.Printf("dial agent err: %v\n", err)
		return
	}
	defer conn.Close()

	agentClient := agent.NewClient(conn)

	// Add key
	err = agentClient.Add(agent.AddedKey{
		PrivateKey: privKey,
		Comment:    "test-comment",
	})
	if err != nil {
		fmt.Printf("agent Add err: %v\n", err)
		return
	}

	fmt.Println("SUCCESS: key added to agent directly!")
}
