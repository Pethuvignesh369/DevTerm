package vault

import (
	"fmt"
	"log"
)

// Vault is the interface for secret storage.
type Vault interface {
	Put(ref string, secret []byte) error
	Get(ref string) ([]byte, error)
	Delete(ref string) error
}

// New creates a Vault implementation. It tries the OS keychain first,
// falling back to an AES-256-GCM encrypted file vault.
func New() (Vault, error) {
	kv, err := newKeychainVault()
	if err == nil {
		log.Println("vault: using OS keychain")
		return kv, nil
	}
	log.Printf("vault: keychain unavailable (%v), using encrypted file vault", err)

	fv, err := newEncryptedFileVault()
	if err != nil {
		return nil, fmt.Errorf("both keychain and file vault failed: %w", err)
	}
	return fv, nil
}
