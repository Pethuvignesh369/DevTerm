package snippetmgr

import (
	"database/sql"
	"fmt"

	"github.com/devterm/core/internal/models"
	"github.com/devterm/core/internal/rpc"
	"github.com/google/uuid"
)

// Manager handles command snippet CRUD.
type Manager struct {
	db *sql.DB
}

// New creates a new snippet manager.
func New(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// RegisterRPC registers snippet RPC methods.
func (m *Manager) RegisterRPC(d *rpc.Dispatcher) {
	d.Register("snippets.list", m.list)
	d.Register("snippets.create", m.create)
	d.Register("snippets.update", m.update)
	d.Register("snippets.delete", m.deleteSnippet)
}

func (m *Manager) list(params map[string]interface{}) (interface{}, error) {
	query, _ := params["query"].(string)

	var rows *sql.Rows
	var err error

	if query != "" {
		pattern := "%" + query + "%"
		rows, err = m.db.Query(
			`SELECT id, title, command, created_at, updated_at FROM snippets WHERE title LIKE ? OR command LIKE ? ORDER BY title`,
			pattern, pattern,
		)
	} else {
		rows, err = m.db.Query(`SELECT id, title, command, created_at, updated_at FROM snippets ORDER BY title`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snippets []models.Snippet
	for rows.Next() {
		var s models.Snippet
		if err := rows.Scan(&s.ID, &s.Title, &s.Command, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		s.Tags = []string{} // TODO: load tags
		snippets = append(snippets, s)
	}
	if snippets == nil {
		snippets = []models.Snippet{}
	}
	return snippets, nil
}

func (m *Manager) create(params map[string]interface{}) (interface{}, error) {
	title, _ := params["title"].(string)
	command, _ := params["command"].(string)
	if title == "" || command == "" {
		return nil, fmt.Errorf("title and command are required")
	}

	id := uuid.New().String()
	_, err := m.db.Exec(
		`INSERT INTO snippets (id, title, command) VALUES (?, ?, ?)`,
		id, title, command,
	)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": id}, nil
}

func (m *Manager) update(params map[string]interface{}) (interface{}, error) {
	id, _ := params["id"].(string)
	title, _ := params["title"].(string)
	command, _ := params["command"].(string)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	_, err := m.db.Exec(
		`UPDATE snippets SET title = COALESCE(NULLIF(?, ''), title), command = COALESCE(NULLIF(?, ''), command), updated_at = datetime('now') WHERE id = ?`,
		title, command, id,
	)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"ok": true}, nil
}

func (m *Manager) deleteSnippet(params map[string]interface{}) (interface{}, error) {
	id, _ := params["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	_, err := m.db.Exec(`DELETE FROM snippets WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"ok": true}, nil
}
