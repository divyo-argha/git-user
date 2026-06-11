//go:build ignore

package main

import (
	"fmt"
	"os"
	"golang.org/x/crypto/ssh"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run test_fingerprint.go <pubkey_path>")
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

	fingerprint := ssh.FingerprintSHA256(pubKey)
	fmt.Printf("Fingerprint: %s\n", fingerprint)
}
