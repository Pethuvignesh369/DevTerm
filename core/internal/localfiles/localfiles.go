package localfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/devterm/core/internal/rpc"
)

// Manager exposes local file metadata to the desktop UI. File contents are
// transferred by the SFTP manager; this service only provides directory views.
type Manager struct{}

func New() *Manager { return &Manager{} }

func (m *Manager) RegisterRPC(d *rpc.Dispatcher) { d.Register("local.list", m.list) }

func (m *Manager) list(params map[string]interface{}) (interface{}, error) {
	path, _ := params["path"].(string)
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("finding home directory: %w", err)
		}
		path = home
	}
	path = filepath.Clean(path)
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("listing local directory: %w", err)
	}

	items := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, map[string]interface{}{
			"name": entry.Name(), "path": filepath.Join(path, entry.Name()),
			"size": info.Size(), "modTime": info.ModTime().Format("2006-01-02T15:04:05Z"),
			"isDir": entry.IsDir(),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		leftDir, rightDir := items[i]["isDir"].(bool), items[j]["isDir"].(bool)
		if leftDir != rightDir {
			return leftDir
		}
		return strings.ToLower(items[i]["name"].(string)) < strings.ToLower(items[j]["name"].(string))
	})
	return map[string]interface{}{"path": path, "entries": items}, nil
}
