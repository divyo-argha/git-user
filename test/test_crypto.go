//go:build ignore

package main

import (
	"errors"
	"fmt"
	"os"
	"golang.org/x/crypto/ssh"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run test_crypto.go <key_path> [passphrase]")
		return
	}
	keyPath := os.Args[1]
	passphrase := ""
	if len(os.Args) > 2 {
		passphrase = os.Args[2]
	}

	data, err := os.ReadFile(keyPath)
	if err != nil {
		fmt.Printf("read file err: %v\n", err)
		return
	}

	// Try without passphrase first
	_, err = ssh.ParseRawPrivateKey(data)
	if err == nil {
		fmt.Println("ParseRawPrivateKey: SUCCESS (unprotected)")
		return
	}
	fmt.Printf("ParseRawPrivateKey err: %v\n", err)

	// Check if it's a passphrase error
	var passphraseErr *ssh.PassphraseMissingError
	if errors.As(err, &passphraseErr) {
		fmt.Println("Is PassphraseMissingError: YES")
	} else {
		fmt.Println("Is PassphraseMissingError: NO")
	}

	// Try with passphrase
	_, err = ssh.ParseRawPrivateKeyWithPassphrase(data, []byte(passphrase))
	if err == nil {
		fmt.Println("ParseRawPrivateKeyWithPassphrase: SUCCESS")
	} else {
		fmt.Printf("ParseRawPrivateKeyWithPassphrase err: %v\n", err)
	}
}
