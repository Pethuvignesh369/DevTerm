package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// encryptedFileVault stores secrets in an AES-256-GCM encrypted JSON file.
type encryptedFileVault struct {
	mu       sync.RWMutex
	filePath string
	key      []byte // 32 bytes for AES-256
	entries  map[string][]byte
}

func newEncryptedFileVault() (*encryptedFileVault, error) {
	vaultPath, err := defaultVaultPath()
	if err != nil {
		return nil, fmt.Errorf("determining vault path: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(vaultPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating vault directory: %w", err)
	}

	// The key is protected with the operating system's data-protection
	// facility when one is available (DPAPI on Windows).
	keyPath := vaultPath + ".key"
	key, err := loadOrCreateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("loading vault key: %w", err)
	}

	v := &encryptedFileVault{
		filePath: vaultPath,
		key:      key,
		entries:  make(map[string][]byte),
	}

	// Load existing data if file exists
	if _, err := os.Stat(vaultPath); err == nil {
		if err := v.load(); err != nil {
			return nil, fmt.Errorf("loading vault data: %w", err)
		}
	}

	return v, nil
}

func (v *encryptedFileVault) Put(ref string, secret []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.entries[ref] = secret
	return v.save()
}

func (v *encryptedFileVault) Get(ref string) ([]byte, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	data, ok := v.entries[ref]
	if !ok {
		return nil, fmt.Errorf("secret not found: %s", ref)
	}
	return data, nil
}

func (v *encryptedFileVault) Delete(ref string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.entries, ref)
	return v.save()
}

func (v *encryptedFileVault) save() error {
	plaintext, err := json.Marshal(v.entries)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(v.key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return os.WriteFile(v.filePath, ciphertext, 0600)
}

func (v *encryptedFileVault) load() error {
	ciphertext, err := os.ReadFile(v.filePath)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(v.key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	return json.Unmarshal(plaintext, &v.entries)
}

func loadOrCreateKey(keyPath string) ([]byte, error) {
	data, err := os.ReadFile(keyPath)
	if err == nil {
		key, unprotectErr := unprotectVaultKey(data)
		if unprotectErr == nil && len(key) == 32 {
			return key, nil
		}

		// Versions before DPAPI stored the key as plaintext. Keep existing
		// installations working and upgrade their key as soon as it is opened.
		if vaultKeyNeedsProtection() && len(data) == 32 {
			if err := writeProtectedKey(keyPath, data); err != nil {
				return nil, fmt.Errorf("upgrading legacy vault key: %w", err)
			}
			return data, nil
		}
		if unprotectErr != nil {
			return nil, fmt.Errorf("unprotecting vault key: %w", unprotectErr)
		}
		return nil, fmt.Errorf("invalid vault key length: got %d bytes", len(key))
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	if err := writeProtectedKey(keyPath, key); err != nil {
		return nil, err
	}
	return key, nil
}

func writeProtectedKey(keyPath string, key []byte) error {
	protected, err := protectVaultKey(key)
	if err != nil {
		return err
	}
	return os.WriteFile(keyPath, protected, 0600)
}

func defaultVaultPath() (string, error) {
	var baseDir string
	switch runtime.GOOS {
	case "windows":
		baseDir = os.Getenv("APPDATA")
		if baseDir == "" {
			baseDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		baseDir = filepath.Join(home, "Library", "Application Support")
	default:
		baseDir = os.Getenv("XDG_DATA_HOME")
		if baseDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			baseDir = filepath.Join(home, ".local", "share")
		}
	}
	return filepath.Join(baseDir, "DevTerm", "vault.dat"), nil
}
