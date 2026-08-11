package settingsmgr

import (
	"database/sql"
	"fmt"

	"github.com/devterm/core/internal/rpc"
)

// Manager persists application preferences shared by the desktop UI and core.
type Manager struct{ db *sql.DB }

func New(db *sql.DB) *Manager { return &Manager{db: db} }

func (m *Manager) RegisterRPC(d *rpc.Dispatcher) {
	d.Register("settings.getAll", m.getAll)
	d.Register("settings.set", m.set)
}

func (m *Manager) getAll(_ map[string]interface{}) (interface{}, error) {
	rows, err := m.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	return settings, rows.Err()
}

func (m *Manager) set(params map[string]interface{}) (interface{}, error) {
	allowed := map[string]bool{
		"theme": true, "terminalTheme": true, "fontFamily": true,
		"fontSize": true, "monitorPollInterval": true, "connectionTimeout": true,
	}
	for key, raw := range params {
		if !allowed[key] {
			continue
		}
		value := fmt.Sprint(raw)
		if _, err := m.db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value); err != nil {
			return nil, err
		}
	}
	return map[string]bool{"ok": true}, nil
}
