// Command hashpassword reads a password from stdin and prints its bcrypt hash.
// It exists so the first user can be created without the password going through a chat
// or a shell history: type it, paste the hash into the INSERT of docs/despliegue.md.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// bcrypt ignores (or rejects) anything after 72 bytes, so longer passwords are refused.
const maxPasswordBytes = 72

func main() {
	os.Exit(run(os.Stdin, os.Stdout, os.Stderr))
}

func run(stdin io.Reader, stdout, stderr io.Writer) int {
	raw, err := io.ReadAll(io.LimitReader(stdin, 4096))
	if err != nil {
		fmt.Fprintln(stderr, "no se pudo leer la contraseña")
		return 1
	}
	// Only the line terminator goes; spaces are part of the password.
	password := strings.TrimSuffix(strings.TrimSuffix(string(raw), "\n"), "\r")
	if password == "" {
		fmt.Fprintln(stderr, "la contraseña está vacía")
		return 1
	}
	if len(password) > maxPasswordBytes {
		fmt.Fprintf(stderr, "la contraseña supera los %d bytes que admite bcrypt\n", maxPasswordBytes)
		return 1
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(stderr, "no se pudo generar el hash")
		return 1
	}
	fmt.Fprintln(stdout, string(hash))
	return 0
}
