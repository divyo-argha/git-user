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
	if len(os.Args) < 2 {
		fmt.Println("usage: go run test_remove.go <pubkey_path>")
		return
	}
	pubPath := os.Args[1]

	data, err := os.ReadFile(pubPath)
	if err != nil {
		fmt.Printf("read file err: %v\n", err)
		return
	}

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(data)
	if err != nil {
		fmt.Printf("ParseAuthorizedKey err: %v\n", err)
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

	err = agentClient.Remove(pubKey)
	if err != nil {
		fmt.Printf("agent Remove err: %v\n", err)
		return
	}

	fmt.Println("SUCCESS: key removed from agent directly!")
}
