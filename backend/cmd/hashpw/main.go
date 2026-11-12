// Command hashpw prints the stored form of a password. It is how the hash in the
// seed data was made:
//
//	go run ./cmd/hashpw 'the-demo-password'
package main

import (
	"fmt"
	"os"

	"staffportal/internal/auth"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: hashpw <password>")
		os.Exit(2)
	}
	hash, err := auth.HashPassword(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "hash failed:", err)
		os.Exit(1)
	}
	fmt.Println(hash)
}
