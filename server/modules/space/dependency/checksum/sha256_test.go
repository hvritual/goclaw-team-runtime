package checksum

import "testing"

func TestSHA256Sum(t *testing.T) {
	const want = "sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got := (SHA256{}).Sum([]byte("hello")); got != want {
		t.Fatalf("Sum = %q, want %q", got, want)
	}
}
