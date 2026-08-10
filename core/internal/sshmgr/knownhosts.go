package sshmgr

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net"

	"golang.org/x/crypto/ssh"
)

// KnownHostsDB implements Trust On First Use (TOFU) host key verification
// using a SQLite database table to store known host keys.
type KnownHostsDB struct {
	db *sql.DB
}

// NewKnownHostsDB creates a new known hosts verifier backed by SQLite.
func NewKnownHostsDB(db *sql.DB) *KnownHostsDB {
	// Ensure the table exists
	db.Exec(`CREATE TABLE IF NOT EXISTS known_hosts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hostname TEXT NOT NULL,
		port INTEGER NOT NULL DEFAULT 22,
		key_type TEXT NOT NULL,
		public_key TEXT NOT NULL,
		fingerprint TEXT NOT NULL,
		first_seen TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(hostname, port, key_type)
	)`)
	return &KnownHostsDB{db: db}
}

// HostKeyCallback returns an ssh.HostKeyCallback that implements TOFU.
// On first connection to a host, the key is stored.
// On subsequent connections, the key is verified against the stored one.
// If the key has changed, the connection is rejected with an error.
func (k *KnownHostsDB) HostKeyCallback() ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// Normalize hostname (strip port from address)
		host, port, err := net.SplitHostPort(hostname)
		if err != nil {
			host = hostname
			port = "22"
		}

		keyType := key.Type()
		publicKeyB64 := base64.StdEncoding.EncodeToString(key.Marshal())
		fingerprint := ssh.FingerprintSHA256(key)

		// Check if we have a stored key for this host
		var storedKey string
		err = k.db.QueryRow(
			`SELECT public_key FROM known_hosts WHERE hostname = ? AND port = ? AND key_type = ?`,
			host, port, keyType,
		).Scan(&storedKey)

		if err == sql.ErrNoRows {
			// First time connecting — TOFU: trust and store
			_, insertErr := k.db.Exec(
				`INSERT INTO known_hosts (hostname, port, key_type, public_key, fingerprint) VALUES (?, ?, ?, ?, ?)`,
				host, port, keyType, publicKeyB64, fingerprint,
			)
			if insertErr != nil {
				log.Printf("[known_hosts] Failed to store key for %s:%s: %v", host, port, insertErr)
				// Don't fail the connection, just log
			} else {
				log.Printf("[known_hosts] TOFU: trusted new key for %s:%s (%s: %s)", host, port, keyType, fingerprint)
			}
			return nil
		}

		if err != nil {
			// DB error — allow connection but log
			log.Printf("[known_hosts] DB error checking %s:%s: %v", host, port, err)
			return nil
		}

		// We have a stored key — verify it matches
		if storedKey != publicKeyB64 {
			return fmt.Errorf(
				"HOST KEY VERIFICATION FAILED for %s:%s\n"+
					"The host key has changed since the first connection.\n"+
					"Expected: %s\n"+
					"Got: %s\n"+
					"This could indicate a man-in-the-middle attack.\n"+
					"If the server was legitimately re-keyed, remove the old key from DevTerm's known hosts.",
				host, port,
				fingerprintStored(k.db, host, port, keyType),
				fingerprint,
			)
		}

		// Key matches — all good
		return nil
	}
}

// RemoveHost removes stored keys for a given host (used when user wants to re-trust).
func (k *KnownHostsDB) RemoveHost(hostname string, port string) error {
	_, err := k.db.Exec(`DELETE FROM known_hosts WHERE hostname = ? AND port = ?`, hostname, port)
	return err
}

// ListHosts returns all known host entries.
func (k *KnownHostsDB) ListHosts() ([]map[string]interface{}, error) {
	rows, err := k.db.Query(`SELECT hostname, port, key_type, fingerprint, first_seen FROM known_hosts ORDER BY hostname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []map[string]interface{}
	for rows.Next() {
		var hostname, keyType, fingerprint, firstSeen string
		var port int
		if err := rows.Scan(&hostname, &port, &keyType, &fingerprint, &firstSeen); err != nil {
			continue
		}
		hosts = append(hosts, map[string]interface{}{
			"hostname":    hostname,
			"port":        port,
			"keyType":     keyType,
			"fingerprint": fingerprint,
			"firstSeen":   firstSeen,
		})
	}
	if hosts == nil {
		hosts = []map[string]interface{}{}
	}
	return hosts, nil
}

func fingerprintStored(db *sql.DB, hostname, port, keyType string) string {
	var fp string
	db.QueryRow(
		`SELECT fingerprint FROM known_hosts WHERE hostname = ? AND port = ? AND key_type = ?`,
		hostname, port, keyType,
	).Scan(&fp)
	return fp
}
