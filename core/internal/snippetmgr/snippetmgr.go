package snippetmgr

import (
	"database/sql"
	"fmt"
	"strings"

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
		if s.Tags, err = m.loadTags(s.ID); err != nil {
			return nil, err
		}
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
	tags := tagNames(params["tags"])
	if title == "" || command == "" {
		return nil, fmt.Errorf("title and command are required")
	}

	id := uuid.New().String()
	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, err = tx.Exec(
		`INSERT INTO snippets (id, title, command) VALUES (?, ?, ?)`,
		id, title, command,
	)
	if err != nil {
		return nil, err
	}
	if err := replaceTags(tx, id, tags); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": id}, nil
}

func (m *Manager) update(params map[string]interface{}) (interface{}, error) {
	id, _ := params["id"].(string)
	title, _ := params["title"].(string)
	command, _ := params["command"].(string)
	tags, tagsProvided := params["tags"]
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, err = tx.Exec(
		`UPDATE snippets SET title = COALESCE(NULLIF(?, ''), title), command = COALESCE(NULLIF(?, ''), command), updated_at = datetime('now') WHERE id = ?`,
		title, command, id,
	)
	if err != nil {
		return nil, err
	}
	if tagsProvided {
		if err := replaceTags(tx, id, tagNames(tags)); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]interface{}{"ok": true}, nil
}

func (m *Manager) loadTags(snippetID string) ([]string, error) {
	rows, err := m.db.Query(`SELECT t.name FROM tags t JOIN snippet_tags st ON st.tag_id = t.id WHERE st.snippet_id = ? ORDER BY t.name`, snippetID)
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

func replaceTags(tx *sql.Tx, snippetID string, tags []string) error {
	if _, err := tx.Exec(`DELETE FROM snippet_tags WHERE snippet_id = ?`, snippetID); err != nil {
		return err
	}
	for _, tag := range tags {
		if _, err := tx.Exec(`INSERT INTO tags (name) VALUES (?) ON CONFLICT(name) DO NOTHING`, tag); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO snippet_tags (snippet_id, tag_id) SELECT ?, id FROM tags WHERE name = ?`, snippetID, tag); err != nil {
			return err
		}
	}
	return nil
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
