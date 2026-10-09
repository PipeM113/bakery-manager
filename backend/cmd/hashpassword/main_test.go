package main

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func hash(t *testing.T, input string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(strings.NewReader(input), &out, &errOut)
	return code, out.String(), errOut.String()
}

// B4: the output is exactly one bcrypt hash that verifies the password.
func TestB4_PrintsOnlyAHashThatVerifiesThePassword(t *testing.T) {
	code, stdout, stderr := hash(t, "una-clave-de-prueba-123\n")

	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr must be empty, got %q", stderr)
	}
	if !strings.HasSuffix(stdout, "\n") || strings.Count(stdout, "\n") != 1 {
		t.Fatalf("stdout must be a single line, got %q", stdout)
	}
	h := strings.TrimSpace(stdout)
	if err := bcrypt.CompareHashAndPassword([]byte(h), []byte("una-clave-de-prueba-123")); err != nil {
		t.Errorf("the hash does not verify the password: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(h), []byte("otra-clave")); err == nil {
		t.Error("the hash must not verify another password")
	}
	if cost, err := bcrypt.Cost([]byte(h)); err != nil || cost < 10 {
		t.Errorf("bcrypt cost = %d (%v), want at least 10", cost, err)
	}
}

// B4: only the line terminator is removed; the rest of the password is kept.
func TestB4_KeepsEverythingButTheLineTerminator(t *testing.T) {
	for name, input := range map[string]string{
		"no newline":   "clave con espacios ",
		"unix newline": "clave con espacios \n",
		"windows crlf": "clave con espacios \r\n",
	} {
		t.Run(name, func(t *testing.T) {
			code, stdout, _ := hash(t, input)

			if code != 0 {
				t.Fatalf("exit code = %d", code)
			}
			if err := bcrypt.CompareHashAndPassword([]byte(strings.TrimSpace(stdout)), []byte("clave con espacios ")); err != nil {
				t.Errorf("the hash should verify the password with its trailing space: %v", err)
			}
		})
	}
}

// B4: the same password gives different hashes (random salt).
func TestB4_EachRunUsesAFreshSalt(t *testing.T) {
	_, first, _ := hash(t, "misma-clave-1234\n")
	_, second, _ := hash(t, "misma-clave-1234\n")

	if first == second {
		t.Fatal("two runs produced the same hash")
	}
}

// B4: empty or too long passwords are refused, with no hash and no echo of the password.
func TestB4_RefusesEmptyAndTooLongPasswords(t *testing.T) {
	cases := []struct {
		name, input, wantInError string
	}{
		{"empty input", "", "vacía"},
		{"only a newline", "\n", "vacía"},
		{"73 bytes", strings.Repeat("a", 73) + "\n", "72"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, stdout, stderr := hash(t, c.input)

			if code == 0 {
				t.Fatal("expected a non-zero exit code")
			}
			if stdout != "" {
				t.Errorf("stdout must stay empty, got %q", stdout)
			}
			if !strings.Contains(stderr, c.wantInError) {
				t.Errorf("stderr = %q, want it to mention %q", stderr, c.wantInError)
			}
			if p := strings.TrimSpace(c.input); p != "" && strings.Contains(stderr, p) {
				t.Error("the error repeats the password")
			}
		})
	}
}

func TestB4_Accepts72BytesExactly(t *testing.T) {
	code, stdout, stderr := hash(t, strings.Repeat("a", 72)+"\n")

	if code != 0 || stdout == "" {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
}
