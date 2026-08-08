package historymgr

import (
	"database/sql"
	"fmt"

	"github.com/devterm/core/internal/models"
	"github.com/devterm/core/internal/rpc"
)

// Manager handles command history recording and search.
type Manager struct {
	db *sql.DB
}

// New creates a new history manager.
func New(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// RegisterRPC registers history RPC methods.
func (m *Manager) RegisterRPC(d *rpc.Dispatcher) {
	d.Register("history.record", m.record)
	d.Register("history.search", m.search)
}

func (m *Manager) record(params map[string]interface{}) (interface{}, error) {
	hostID, _ := params["hostId"].(string)
	command, _ := params["command"].(string)
	if hostID == "" || command == "" {
		return nil, fmt.Errorf("hostId and command are required")
	}

	_, err := m.db.Exec(
		`INSERT INTO command_history (host_id, command) VALUES (?, ?)`,
		hostID, command,
	)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"ok": true}, nil
}

func (m *Manager) search(params map[string]interface{}) (interface{}, error) {
	hostID, _ := params["hostId"].(string)
	query, _ := params["query"].(string)
	limit := 50
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}
	offset := 0
	if o, ok := params["offset"].(float64); ok {
		offset = int(o)
	}

	var rows *sql.Rows
	var err error

	if hostID != "" && query != "" {
		pattern := "%" + query + "%"
		rows, err = m.db.Query(
			`SELECT id, host_id, command, executed_at FROM command_history WHERE host_id = ? AND command LIKE ? ORDER BY executed_at DESC LIMIT ? OFFSET ?`,
			hostID, pattern, limit, offset,
		)
	} else if hostID != "" {
		rows, err = m.db.Query(
			`SELECT id, host_id, command, executed_at FROM command_history WHERE host_id = ? ORDER BY executed_at DESC LIMIT ? OFFSET ?`,
			hostID, limit, offset,
		)
	} else if query != "" {
		pattern := "%" + query + "%"
		rows, err = m.db.Query(
			`SELECT id, host_id, command, executed_at FROM command_history WHERE command LIKE ? ORDER BY executed_at DESC LIMIT ? OFFSET ?`,
			pattern, limit, offset,
		)
	} else {
		rows, err = m.db.Query(
			`SELECT id, host_id, command, executed_at FROM command_history ORDER BY executed_at DESC LIMIT ? OFFSET ?`,
			limit, offset,
		)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.HistoryEntry
	for rows.Next() {
		var e models.HistoryEntry
		if err := rows.Scan(&e.ID, &e.HostID, &e.Command, &e.ExecutedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []models.HistoryEntry{}
	}
	return entries, nil
}
