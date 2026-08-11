package keymgr

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/pem"
	"fmt"

	"github.com/devterm/core/internal/models"
	"github.com/devterm/core/internal/rpc"
	"github.com/devterm/core/internal/vault"
	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

// Manager handles SSH key generation, import, and management.
type Manager struct {
	db    *sql.DB
	vault vault.Vault
}

// New creates a new key manager.
func New(db *sql.DB, v vault.Vault) *Manager {
	return &Manager{db: db, vault: v}
}

// RegisterRPC registers key management RPC methods.
func (m *Manager) RegisterRPC(d *rpc.Dispatcher) {
	d.Register("keys.generate", m.generate)
	d.Register("keys.import", m.importKey)
	d.Register("keys.list", m.list)
	d.Register("keys.delete", m.deleteKey)
}

// GetKey retrieves key metadata by ID.
func (m *Manager) GetKey(id string) (*models.SSHKey, error) {
	row := m.db.QueryRow(`SELECT id, name, key_type, public_key, fingerprint, passphrase_protected, vault_ref, passphrase_vault_ref, created_at FROM ssh_keys WHERE id = ?`, id)
	var k models.SSHKey
	var passphraseRef sql.NullString
	err := row.Scan(&k.ID, &k.Name, &k.KeyType, &k.PublicKey, &k.Fingerprint, &k.PassphraseProtected, &k.VaultRef, &passphraseRef, &k.CreatedAt)
	if err != nil {
		return nil, err
	}
	if passphraseRef.Valid {
		k.PassphraseVaultRef = &passphraseRef.String
	}
	return &k, nil
}

func (m *Manager) generate(params map[string]interface{}) (interface{}, error) {
	name, _ := params["name"].(string)
	keyType, _ := params["keyType"].(string)
	passphrase, _ := params["passphrase"].(string)

	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if keyType == "" {
		keyType = "ed25519"
	}

	var privateKeyBytes []byte
	var pubKey ssh.PublicKey

	switch keyType {
	case "rsa":
		bits := 4096
		if b, ok := params["bits"].(float64); ok {
			bits = int(b)
		}
		rsaKey, err := rsa.GenerateKey(rand.Reader, bits)
		if err != nil {
			return nil, fmt.Errorf("generating RSA key: %w", err)
		}
		signer, err := ssh.NewSignerFromKey(rsaKey)
		if err != nil {
			return nil, err
		}
		pubKey = signer.PublicKey()
		// Marshal private key to PEM
		var pemBlock *pem.Block
		if passphrase != "" {
			pemBlock, err = ssh.MarshalPrivateKeyWithPassphrase(rsaKey, "", []byte(passphrase))
		} else {
			pemBlock, err = ssh.MarshalPrivateKey(rsaKey, "")
		}
		if err != nil {
			return nil, fmt.Errorf("marshaling private key: %w", err)
		}
		privateKeyBytes = pem.EncodeToMemory(pemBlock)

	case "ed25519":
		_, privKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generating ED25519 key: %w", err)
		}
		signer, err := ssh.NewSignerFromKey(privKey)
		if err != nil {
			return nil, err
		}
		pubKey = signer.PublicKey()
		var pemBlock *pem.Block
		if passphrase != "" {
			pemBlock, err = ssh.MarshalPrivateKeyWithPassphrase(privKey, "", []byte(passphrase))
		} else {
			pemBlock, err = ssh.MarshalPrivateKey(privKey, "")
		}
		if err != nil {
			return nil, fmt.Errorf("marshaling private key: %w", err)
		}
		privateKeyBytes = pem.EncodeToMemory(pemBlock)

	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}

	// Store in vault
	vaultRef := uuid.New().String()
	if err := m.vault.Put(vaultRef, privateKeyBytes); err != nil {
		return nil, fmt.Errorf("storing key in vault: %w", err)
	}
	var passphraseRef *string
	if passphrase != "" {
		ref := uuid.New().String()
		if err := m.vault.Put(ref, []byte(passphrase)); err != nil {
			return nil, fmt.Errorf("storing key passphrase: %w", err)
		}
		passphraseRef = &ref
	}

	// Store metadata in DB
	id := uuid.New().String()
	fingerprint := ssh.FingerprintSHA256(pubKey)
	publicKeyStr := string(ssh.MarshalAuthorizedKey(pubKey))
	passphraseProtected := 0
	if passphrase != "" {
		passphraseProtected = 1
	}

	_, err := m.db.Exec(
		`INSERT INTO ssh_keys (id, name, key_type, public_key, fingerprint, passphrase_protected, vault_ref, passphrase_vault_ref) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, name, keyType, publicKeyStr, fingerprint, passphraseProtected, vaultRef, passphraseRef,
	)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":          id,
		"name":        name,
		"keyType":     keyType,
		"fingerprint": fingerprint,
		"publicKey":   publicKeyStr,
	}, nil
}

func (m *Manager) importKey(params map[string]interface{}) (interface{}, error) {
	name, _ := params["name"].(string)
	privateKeyPEM, _ := params["privateKey"].(string)
	passphrase, _ := params["passphrase"].(string)

	if name == "" || privateKeyPEM == "" {
		return nil, fmt.Errorf("name and privateKey are required")
	}

	// Parse to validate
	signer, err := ssh.ParsePrivateKey([]byte(privateKeyPEM))
	if err != nil {
		// Try with passphrase
		if passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(privateKeyPEM), []byte(passphrase))
		}
		if err != nil {
			return nil, fmt.Errorf("invalid private key: %w", err)
		}
	}

	pubKey := signer.PublicKey()
	fingerprint := ssh.FingerprintSHA256(pubKey)
	publicKeyStr := string(ssh.MarshalAuthorizedKey(pubKey))

	// Determine key type
	keyType := pubKey.Type()
	switch keyType {
	case "ssh-rsa":
		keyType = "rsa"
	case "ssh-ed25519":
		keyType = "ed25519"
	}

	// Store key and, when necessary, its passphrase in separate vault entries.
	vaultRef := uuid.New().String()
	if err := m.vault.Put(vaultRef, []byte(privateKeyPEM)); err != nil {
		return nil, fmt.Errorf("storing key in vault: %w", err)
	}

	var passphraseRef *string
	if passphrase != "" {
		ref := uuid.New().String()
		if err := m.vault.Put(ref, []byte(passphrase)); err != nil {
			return nil, fmt.Errorf("storing key passphrase: %w", err)
		}
		passphraseRef = &ref
	}
	id := uuid.New().String()
	_, err = m.db.Exec(
		`INSERT INTO ssh_keys (id, name, key_type, public_key, fingerprint, passphrase_protected, vault_ref, passphrase_vault_ref) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, name, keyType, publicKeyStr, fingerprint, passphrase != "", vaultRef, passphraseRef,
	)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":          id,
		"name":        name,
		"keyType":     keyType,
		"fingerprint": fingerprint,
		"publicKey":   publicKeyStr,
	}, nil
}

func (m *Manager) list(params map[string]interface{}) (interface{}, error) {
	rows, err := m.db.Query(`SELECT id, name, key_type, public_key, fingerprint, passphrase_protected, vault_ref, passphrase_vault_ref, created_at FROM ssh_keys ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []models.SSHKey
	for rows.Next() {
		var k models.SSHKey
		var passphraseRef sql.NullString
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyType, &k.PublicKey, &k.Fingerprint, &k.PassphraseProtected, &k.VaultRef, &passphraseRef, &k.CreatedAt); err != nil {
			return nil, err
		}
		if passphraseRef.Valid {
			k.PassphraseVaultRef = &passphraseRef.String
		}
		keys = append(keys, k)
	}
	if keys == nil {
		keys = []models.SSHKey{}
	}
	return keys, nil
}

func (m *Manager) deleteKey(params map[string]interface{}) (interface{}, error) {
	id, _ := params["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	// Get vault ref to delete secret
	var vaultRef string
	var passphraseRef sql.NullString
	err := m.db.QueryRow(`SELECT vault_ref, passphrase_vault_ref FROM ssh_keys WHERE id = ?`, id).Scan(&vaultRef, &passphraseRef)
	if err == nil && vaultRef != "" {
		_ = m.vault.Delete(vaultRef)
	}
	if passphraseRef.Valid {
		_ = m.vault.Delete(passphraseRef.String)
	}

	_, err = m.db.Exec(`DELETE FROM ssh_keys WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"ok": true}, nil
}
