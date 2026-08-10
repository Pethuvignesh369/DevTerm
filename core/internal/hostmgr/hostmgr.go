package hostmgr

import (
	"database/sql"
	"fmt"

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
func New(db *sql.DB, vault ...interface{ Put(ref string, secret []byte) error }) *Manager {
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
		h.Tags = []string{} // TODO: load tags
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
	username, _ := params["username"].(string)
	identityID, _ := params["identityId"].(string)
	favorite := false
	if f, ok := params["favorite"].(bool); ok {
		favorite = f
	}

	var identityIDPtr *string
	if identityID != "" {
		identityIDPtr = &identityID
	}

	_, err := m.db.Exec(
		`INSERT INTO hosts (id, name, hostname, port, username, identity_id, favorite) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, name, hostname, port, username, identityIDPtr, favorite,
	)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": id}, nil
}

func (m *Manager) update(params map[string]interface{}) (interface{}, error) {
	id, _ := params["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	// Build SET clause dynamically
	name, _ := params["name"].(string)
	hostname, _ := params["hostname"].(string)
	username, _ := params["username"].(string)

	_, err := m.db.Exec(
		`UPDATE hosts SET name = COALESCE(NULLIF(?, ''), name), hostname = COALESCE(NULLIF(?, ''), hostname), username = COALESCE(NULLIF(?, ''), username), updated_at = datetime('now') WHERE id = ?`,
		name, hostname, username, id,
	)
	if err != nil {
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
		h.Tags = []string{}
		hosts = append(hosts, h)
	}
	if hosts == nil {
		hosts = []models.Host{}
	}
	return hosts, nil
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
