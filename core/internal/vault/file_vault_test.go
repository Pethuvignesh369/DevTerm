package vault

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateKeyPersistsProtectedKey(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "vault.key")

	first, err := loadOrCreateKey(keyPath)
	if err != nil {
		t.Fatalf("creating key: %v", err)
	}
	second, err := loadOrCreateKey(keyPath)
	if err != nil {
		t.Fatalf("reloading key: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("reloaded key does not match generated key")
	}

	stored, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("reading stored key: %v", err)
	}
	if vaultKeyNeedsProtection() && bytes.Equal(stored, first) {
		t.Fatal("protected vault key was written as plaintext")
	}
}

func TestLoadOrCreateKeyMigratesLegacyWindowsKey(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "vault.key")
	legacy := bytes.Repeat([]byte{0x5a}, 32)
	if err := os.WriteFile(keyPath, legacy, 0600); err != nil {
		t.Fatalf("writing legacy key: %v", err)
	}

	key, err := loadOrCreateKey(keyPath)
	if err != nil {
		t.Fatalf("loading legacy key: %v", err)
	}
	if !bytes.Equal(key, legacy) {
		t.Fatal("legacy key changed during migration")
	}
	if vaultKeyNeedsProtection() {
		stored, err := os.ReadFile(keyPath)
		if err != nil {
			t.Fatalf("reading migrated key: %v", err)
		}
		if bytes.Equal(stored, legacy) {
			t.Fatal("legacy key was not migrated to OS protection")
		}
	}
}

func TestLoadOrCreateKeyRejectsCorruption(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "vault.key")
	if err := os.WriteFile(keyPath, []byte("corrupt"), 0600); err != nil {
		t.Fatalf("writing corrupt key: %v", err)
	}
	if _, err := loadOrCreateKey(keyPath); err == nil {
		t.Fatal("corrupt key was silently replaced")
	}
}
