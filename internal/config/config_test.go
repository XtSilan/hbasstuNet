package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoadRememberedPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	want := Settings{Username: "student", Password: "secret-value", Role: "student", ISP: "cucc", Remember: true}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), want.Password) {
		t.Fatal("password was stored as plaintext")
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Password != want.Password {
		t.Fatalf("password round trip = %q, want %q", got.Password, want.Password)
	}
}

func TestSaveWithoutRememberingClearsPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := Save(path, Settings{Password: "secret-value", Role: "student", Remember: false}); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Password != "" {
		t.Fatalf("password = %q, want empty", got.Password)
	}
}
