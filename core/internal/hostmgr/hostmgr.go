package hostmgr

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/devterm/core/internal/models"
	"github.com/devterm/core/internal/rpc"
	"github.com/google/uuid"
)

// Manager handles host and identity CRUD.
type Manager struct {
	db    *sql.DB
	vault interface {
		Put(ref string, secret []byte) error
	}
}

// New creates a new host manager.
func New(db *sql.DB, vault ...interface {
	Put(ref string, secret []byte) error
}) *Manager {
	m := &Manager{db: db}
	if len(vault) > 0 {
		m.vault = vault[0]
	}
	return m
}

// RegisterRPC registers all host management RPC methods.
func (m *Manager) RegisterRPC(d *rpc.Dispatcher) {
	d.Register("hosts.list", m.list)
	d.Register("hosts.create", m.create)
	d.Register("hosts.update", m.update)
	d.Register("hosts.delete", m.deleteHost)
	d.Register("hosts.search", m.search)
	d.Register("hosts.importSSHConfig", m.importSSHConfig)
	d.Register("hosts.export", m.exportHosts)
	d.Register("hosts.import", m.importHosts)
	d.Register("identities.list", m.listIdentities)
	d.Register("identities.create", m.createIdentity)
	d.Register("identities.delete", m.deleteIdentity)
	d.Register("groups.list", m.listGroups)
	d.Register("groups.create", m.createGroup)
	d.Register("groups.delete", m.deleteGroup)
}

// GetHost retrieves a host by ID (used by other managers).
func (m *Manager) GetHost(id string) (*models.Host, error) {
	row := m.db.QueryRow(`SELECT id, name, hostname, port, username, identity_id, group_id, favorite, created_at, updated_at FROM hosts WHERE id = ?`, id)
	var h models.Host
	var identityID, groupID sql.NullString
	var groupIDInt sql.NullInt64
	err := row.Scan(&h.ID, &h.Name, &h.Hostname, &h.Port, &h.Username, &identityID, &groupIDInt, &h.Favorite, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("host not found: %w", err)
	}
	if identityID.Valid {
		h.IdentityID = &identityID.String
	}
	if groupIDInt.Valid {
		gid := int(groupIDInt.Int64)
		h.GroupID = &gid
	}
	_ = groupID // suppress unused
	if h.Tags, err = m.loadTags(h.ID); err != nil {
		return nil, err
	}
	return &h, nil
}

// GetIdentity retrieves an identity by ID.
func (m *Manager) GetIdentity(id string) (*models.Identity, error) {
	row := m.db.QueryRow(`SELECT id, name, auth_type, ssh_key_id, vault_ref, created_at FROM identities WHERE id = ?`, id)
	var ident models.Identity
	var sshKeyID, vaultRef sql.NullString
	err := row.Scan(&ident.ID, &ident.Name, &ident.AuthType, &sshKeyID, &vaultRef, &ident.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("identity not found: %w", err)
	}
	if sshKeyID.Valid {
		ident.SSHKeyID = &sshKeyID.String
	}
	if vaultRef.Valid {
		ident.VaultRef = &vaultRef.String
	}
	return &ident, nil
}

func (m *Manager) list(params map[string]interface{}) (interface{}, error) {
	rows, err := m.db.Query(`SELECT id, name, hostname, port, username, identity_id, group_id, favorite, created_at, updated_at FROM hosts ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []models.Host
	for rows.Next() {
		var h models.Host
		var identityID sql.NullString
		var groupID sql.NullInt64
		if err := rows.Scan(&h.ID, &h.Name, &h.Hostname, &h.Port, &h.Username, &identityID, &groupID, &h.Favorite, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		if identityID.Valid {
			h.IdentityID = &identityID.String
		}
		if groupID.Valid {
			gid := int(groupID.Int64)
			h.GroupID = &gid
		}
		if h.Tags, err = m.loadTags(h.ID); err != nil {
			return nil, err
		}
		hosts = append(hosts, h)
	}
	if hosts == nil {
		hosts = []models.Host{}
	}
	return hosts, nil
}

func (m *Manager) create(params map[string]interface{}) (interface{}, error) {
	id := uuid.New().String()
	name, _ := params["name"].(string)
	hostname, _ := params["hostname"].(string)
	port := 22
	if p, ok := params["port"].(float64); ok {
		port = int(p)
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("port must be between 1 and 65535")
	}
	username, _ := params["username"].(string)
	if name == "" || hostname == "" || username == "" {
		return nil, fmt.Errorf("name, hostname, and username are required")
	}
	identityID, _ := params["identityId"].(string)
	tags := tagNames(params["tags"])
	favorite := false
	if f, ok := params["favorite"].(bool); ok {
		favorite = f
	}

	var identityIDPtr *string
	if identityID != "" {
		identityIDPtr = &identityID
	}

	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, err = tx.Exec(
		`INSERT INTO hosts (id, name, hostname, port, username, identity_id, favorite) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, name, hostname, port, username, identityIDPtr, favorite,
	)
	if err != nil {
		return nil, err
	}
	if err := replaceTags(tx, "host_tags", "host_id", id, tags); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": id}, nil
}

func (m *Manager) update(params map[string]interface{}) (interface{}, error) {
	id, _ := params["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	// Keep omitted fields intact while allowing every editable host field to persist.
	name, _ := params["name"].(string)
	hostname, _ := params["hostname"].(string)
	username, _ := params["username"].(string)
	port := 0
	if raw, ok := params["port"].(float64); ok {
		port = int(raw)
	}
	if port < 0 || port > 65535 {
		return nil, fmt.Errorf("port must be between 1 and 65535")
	}
	var favorite interface{}
	if value, ok := params["favorite"].(bool); ok {
		favorite = value
	}
	var identityID interface{}
	if value, ok := params["identityId"].(string); ok {
		identityID = value
	}
	var groupID interface{}
	if value, ok := params["groupId"].(float64); ok {
		groupID = int(value)
	}
	tags, tagsProvided := params["tags"]

	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, err = tx.Exec(
		`UPDATE hosts SET
		 name = COALESCE(NULLIF(?, ''), name), hostname = COALESCE(NULLIF(?, ''), hostname),
		 username = COALESCE(NULLIF(?, ''), username), port = CASE WHEN ? > 0 THEN ? ELSE port END,
		 favorite = COALESCE(?, favorite), identity_id = COALESCE(NULLIF(?, ''), identity_id),
		 group_id = COALESCE(?, group_id), updated_at = datetime('now') WHERE id = ?`,
		name, hostname, username, port, port, favorite, identityID, groupID, id,
	)
	if err != nil {
		return nil, err
	}
	if tagsProvided {
		if err := replaceTags(tx, "host_tags", "host_id", id, tagNames(tags)); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]interface{}{"ok": true}, nil
}

func (m *Manager) deleteHost(params map[string]interface{}) (interface{}, error) {
	id, _ := params["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	_, err := m.db.Exec(`DELETE FROM hosts WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"ok": true}, nil
}

func (m *Manager) search(params map[string]interface{}) (interface{}, error) {
	query, _ := params["query"].(string)
	if query == "" {
		return m.list(params)
	}
	pattern := "%" + query + "%"
	rows, err := m.db.Query(
		`SELECT id, name, hostname, port, username, identity_id, group_id, favorite, created_at, updated_at FROM hosts WHERE name LIKE ? OR hostname LIKE ? ORDER BY name`,
		pattern, pattern,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []models.Host
	for rows.Next() {
		var h models.Host
		var identityID sql.NullString
		var groupID sql.NullInt64
		if err := rows.Scan(&h.ID, &h.Name, &h.Hostname, &h.Port, &h.Username, &identityID, &groupID, &h.Favorite, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		if identityID.Valid {
			h.IdentityID = &identityID.String
		}
		if groupID.Valid {
			gid := int(groupID.Int64)
			h.GroupID = &gid
		}
		if h.Tags, err = m.loadTags(h.ID); err != nil {
			return nil, err
		}
		hosts = append(hosts, h)
	}
	if hosts == nil {
		hosts = []models.Host{}
	}
	return hosts, nil
}

func (m *Manager) loadTags(hostID string) ([]string, error) {
	rows, err := m.db.Query(`SELECT t.name FROM tags t JOIN host_tags ht ON ht.tag_id = t.id WHERE ht.host_id = ? ORDER BY t.name`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := []string{}
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func tagNames(raw interface{}) []string {
	values, ok := raw.([]interface{})
	if !ok {
		return []string{}
	}
	seen, tags := make(map[string]struct{}), make([]string, 0, len(values))
	for _, value := range values {
		name, ok := value.(string)
		name = strings.ToLower(strings.TrimSpace(name))
		if !ok || name == "" {
			continue
		}
		if _, exists := seen[name]; !exists {
			seen[name] = struct{}{}
			tags = append(tags, name)
		}
	}
	return tags
}

func replaceTags(tx *sql.Tx, relation, ownerColumn, ownerID string, tags []string) error {
	if _, err := tx.Exec(`DELETE FROM `+relation+` WHERE `+ownerColumn+` = ?`, ownerID); err != nil {
		return err
	}
	for _, tag := range tags {
		if _, err := tx.Exec(`INSERT INTO tags (name) VALUES (?) ON CONFLICT(name) DO NOTHING`, tag); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO `+relation+` (`+ownerColumn+`, tag_id) SELECT ?, id FROM tags WHERE name = ?`, ownerID, tag); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) listIdentities(params map[string]interface{}) (interface{}, error) {
	rows, err := m.db.Query(`SELECT id, name, auth_type, ssh_key_id, vault_ref, created_at FROM identities ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var identities []models.Identity
	for rows.Next() {
		var i models.Identity
		var sshKeyID, vaultRef sql.NullString
		if err := rows.Scan(&i.ID, &i.Name, &i.AuthType, &sshKeyID, &vaultRef, &i.CreatedAt); err != nil {
			return nil, err
		}
		if sshKeyID.Valid {
			i.SSHKeyID = &sshKeyID.String
		}
		if vaultRef.Valid {
			i.VaultRef = &vaultRef.String
		}
		identities = append(identities, i)
	}
	if identities == nil {
		identities = []models.Identity{}
	}
	return identities, nil
}

func (m *Manager) createIdentity(params map[string]interface{}) (interface{}, error) {
	id := uuid.New().String()
	name, _ := params["name"].(string)
	authType, _ := params["authType"].(string)
	sshKeyID, _ := params["sshKeyId"].(string)
	vaultRef, _ := params["vaultRef"].(string)
	password, _ := params["password"].(string)
	if name == "" {
		return nil, fmt.Errorf("identity name is required")
	}
	if authType != "password" && authType != "key" && authType != "agent" {
		return nil, fmt.Errorf("unsupported authentication type: %s", authType)
	}
	if authType == "password" && password == "" {
		return nil, fmt.Errorf("password is required for password authentication")
	}
	if authType == "key" && sshKeyID == "" {
		return nil, fmt.Errorf("sshKeyId is required for key authentication")
	}

	// If password auth, store the password in vault
	if authType == "password" && password != "" && m.vault != nil {
		vaultRef = uuid.New().String()
		if err := m.vault.Put(vaultRef, []byte(password)); err != nil {
			return nil, fmt.Errorf("failed to store password: %w", err)
		}
	}

	var sshKeyIDPtr, vaultRefPtr *string
	if sshKeyID != "" {
		sshKeyIDPtr = &sshKeyID
	}
	if vaultRef != "" {
		vaultRefPtr = &vaultRef
	}

	_, err := m.db.Exec(
		`INSERT INTO identities (id, name, auth_type, ssh_key_id, vault_ref) VALUES (?, ?, ?, ?, ?)`,
		id, name, authType, sshKeyIDPtr, vaultRefPtr,
	)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": id}, nil
}

func (m *Manager) deleteIdentity(params map[string]interface{}) (interface{}, error) {
	id, _ := params["id"].(string)
	_, err := m.db.Exec(`DELETE FROM identities WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"ok": true}, nil
}

func (m *Manager) listGroups(params map[string]interface{}) (interface{}, error) {
	rows, err := m.db.Query(`SELECT id, name, created_at FROM groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []models.Group
	for rows.Next() {
		var g models.Group
		if err := rows.Scan(&g.ID, &g.Name, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	if groups == nil {
		groups = []models.Group{}
	}
	return groups, nil
}

func (m *Manager) createGroup(params map[string]interface{}) (interface{}, error) {
	name, _ := params["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	result, err := m.db.Exec(`INSERT INTO groups (name) VALUES (?)`, name)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return map[string]interface{}{"id": id}, nil
}

func (m *Manager) deleteGroup(params map[string]interface{}) (interface{}, error) {
	id, ok := params["id"].(float64)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}
	_, err := m.db.Exec(`DELETE FROM groups WHERE id = ?`, int(id))
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"ok": true}, nil
}

func (m *Manager) importSSHConfig(params map[string]interface{}) (interface{}, error) {
	entries, err := ParseSSHConfig()
	if err != nil {
		return nil, err
	}

	imported := 0
	for _, entry := range entries {
		// Check if host already exists with same hostname
		var exists int
		m.db.QueryRow(`SELECT COUNT(*) FROM hosts WHERE hostname = ? AND port = ?`, entry.Hostname, entry.Port).Scan(&exists)
		if exists > 0 {
			continue
		}

		id := uuid.New().String()
		_, err := m.db.Exec(
			`INSERT INTO hosts (id, name, hostname, port, username) VALUES (?, ?, ?, ?, ?)`,
			id, entry.Name, entry.Hostname, entry.Port, entry.Username,
		)
		if err == nil {
			imported++
		}
	}

	return map[string]interface{}{
		"imported": imported,
		"total":    len(entries),
	}, nil
}

func (m *Manager) exportHosts(params map[string]interface{}) (interface{}, error) {
	hosts, err := m.list(params)
	if err != nil {
		return nil, err
	}
	identities, err := m.listIdentities(params)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"hosts":      hosts,
		"identities": identities,
		"exportedAt": "now",
	}, nil
}

func (m *Manager) importHosts(params map[string]interface{}) (interface{}, error) {
	hostsRaw, ok := params["hosts"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("hosts array is required")
	}

	imported := 0
	for _, h := range hostsRaw {
		hostMap, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		_, err := m.create(hostMap)
		if err == nil {
			imported++
		}
	}
	return map[string]interface{}{"imported": imported}, nil
}
