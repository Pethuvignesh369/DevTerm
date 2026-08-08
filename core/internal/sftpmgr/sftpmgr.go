package sftpmgr

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/devterm/core/internal/rpc"
	"github.com/devterm/core/internal/sshmgr"
	"github.com/google/uuid"
	"github.com/pkg/sftp"
)

// Manager handles SFTP file operations.
type Manager struct {
	sshMgr *sshmgr.Manager
}

// New creates a new SFTP manager.
func New(sshMgr *sshmgr.Manager) *Manager {
	return &Manager{sshMgr: sshMgr}
}

// RegisterRPC registers SFTP RPC methods.
func (m *Manager) RegisterRPC(d *rpc.Dispatcher) {
	d.Register("sftp.list", m.list)
	d.Register("sftp.upload", m.upload)
	d.Register("sftp.download", m.download)
	d.Register("sftp.rename", m.rename)
	d.Register("sftp.delete", m.deleteFile)
	d.Register("sftp.mkdir", m.mkdir)
}

func (m *Manager) getClient(sessionID string) (*sftp.Client, error) {
	sshClient, err := m.sshMgr.GetClient(sessionID)
	if err != nil {
		return nil, err
	}
	return sftp.NewClient(sshClient)
}

func (m *Manager) list(params map[string]interface{}) (interface{}, error) {
	sessionID, _ := params["sessionId"].(string)
	path, _ := params["path"].(string)
	if sessionID == "" {
		return nil, fmt.Errorf("sessionId is required")
	}
	if path == "" {
		path = "."
	}

	client, err := m.getClient(sessionID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	entries, err := client.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("listing directory: %w", err)
	}

	var items []map[string]interface{}
	for _, entry := range entries {
		items = append(items, map[string]interface{}{
			"name":    entry.Name(),
			"size":    entry.Size(),
			"mode":    entry.Mode().String(),
			"modTime": entry.ModTime().Format("2006-01-02T15:04:05Z"),
			"isDir":   entry.IsDir(),
		})
	}
	if items == nil {
		items = []map[string]interface{}{}
	}
	return items, nil
}

func (m *Manager) upload(params map[string]interface{}) (interface{}, error) {
	sessionID, _ := params["sessionId"].(string)
	localPath, _ := params["localPath"].(string)
	remotePath, _ := params["remotePath"].(string)

	if sessionID == "" || localPath == "" || remotePath == "" {
		return nil, fmt.Errorf("sessionId, localPath, and remotePath are required")
	}

	client, err := m.getClient(sessionID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	transferId := uuid.New().String()
	notifier := rpc.GetNotifier()

	// Open local file
	localFile, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("opening local file: %w", err)
	}

	stat, err := localFile.Stat()
	if err != nil {
		localFile.Close()
		return nil, fmt.Errorf("stat local file: %w", err)
	}
	totalSize := stat.Size()

	// Create remote file
	remoteFile, err := client.Create(remotePath)
	if err != nil {
		localFile.Close()
		return nil, fmt.Errorf("creating remote file: %w", err)
	}

	// Copy in chunks with progress
	go func() {
		defer localFile.Close()
		defer remoteFile.Close()

		buf := make([]byte, 32*1024)
		var written int64

		for {
			n, readErr := localFile.Read(buf)
			if n > 0 {
				_, writeErr := remoteFile.Write(buf[:n])
				if writeErr != nil {
					if notifier != nil {
						notifier.Notify("sftp.progress", map[string]interface{}{
							"transferId": transferId,
							"percent":    0,
							"status":     "failed",
						})
					}
					return
				}
				written += int64(n)
				if notifier != nil && totalSize > 0 {
					percent := int(float64(written) / float64(totalSize) * 100)
					notifier.Notify("sftp.progress", map[string]interface{}{
						"transferId": transferId,
						"percent":    percent,
						"status":     "in_progress",
					})
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				if notifier != nil {
					notifier.Notify("sftp.progress", map[string]interface{}{
						"transferId": transferId,
						"percent":    0,
						"status":     "failed",
					})
				}
				return
			}
		}

		if notifier != nil {
			notifier.Notify("sftp.complete", map[string]interface{}{
				"transferId": transferId,
			})
		}
	}()

	return map[string]interface{}{"transferId": transferId}, nil
}

func (m *Manager) download(params map[string]interface{}) (interface{}, error) {
	sessionID, _ := params["sessionId"].(string)
	remotePath, _ := params["remotePath"].(string)
	localPath, _ := params["localPath"].(string)

	if sessionID == "" || remotePath == "" || localPath == "" {
		return nil, fmt.Errorf("sessionId, remotePath, and localPath are required")
	}

	client, err := m.getClient(sessionID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	transferId := uuid.New().String()
	notifier := rpc.GetNotifier()

	// Open remote file
	remoteFile, err := client.Open(remotePath)
	if err != nil {
		return nil, fmt.Errorf("opening remote file: %w", err)
	}

	stat, err := remoteFile.Stat()
	if err != nil {
		remoteFile.Close()
		return nil, fmt.Errorf("stat remote file: %w", err)
	}
	totalSize := stat.Size()

	// Ensure local directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		remoteFile.Close()
		return nil, fmt.Errorf("creating local directory: %w", err)
	}

	// Create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		remoteFile.Close()
		return nil, fmt.Errorf("creating local file: %w", err)
	}

	// Copy in chunks with progress
	go func() {
		defer remoteFile.Close()
		defer localFile.Close()

		buf := make([]byte, 32*1024)
		var written int64

		for {
			n, readErr := remoteFile.Read(buf)
			if n > 0 {
				_, writeErr := localFile.Write(buf[:n])
				if writeErr != nil {
					if notifier != nil {
						notifier.Notify("sftp.progress", map[string]interface{}{
							"transferId": transferId,
							"percent":    0,
							"status":     "failed",
						})
					}
					return
				}
				written += int64(n)
				if notifier != nil && totalSize > 0 {
					percent := int(float64(written) / float64(totalSize) * 100)
					notifier.Notify("sftp.progress", map[string]interface{}{
						"transferId": transferId,
						"percent":    percent,
						"status":     "in_progress",
					})
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				if notifier != nil {
					notifier.Notify("sftp.progress", map[string]interface{}{
						"transferId": transferId,
						"percent":    0,
						"status":     "failed",
					})
				}
				return
			}
		}

		if notifier != nil {
			notifier.Notify("sftp.complete", map[string]interface{}{
				"transferId": transferId,
			})
		}
	}()

	return map[string]interface{}{"transferId": transferId}, nil
}

func (m *Manager) rename(params map[string]interface{}) (interface{}, error) {
	sessionID, _ := params["sessionId"].(string)
	oldPath, _ := params["oldPath"].(string)
	newPath, _ := params["newPath"].(string)

	if sessionID == "" || oldPath == "" || newPath == "" {
		return nil, fmt.Errorf("sessionId, oldPath, and newPath are required")
	}

	client, err := m.getClient(sessionID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if err := client.Rename(oldPath, newPath); err != nil {
		return nil, fmt.Errorf("rename failed: %w", err)
	}
	return map[string]interface{}{"ok": true}, nil
}

func (m *Manager) deleteFile(params map[string]interface{}) (interface{}, error) {
	sessionID, _ := params["sessionId"].(string)
	path, _ := params["path"].(string)

	if sessionID == "" || path == "" {
		return nil, fmt.Errorf("sessionId and path are required")
	}

	client, err := m.getClient(sessionID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if err := client.Remove(path); err != nil {
		return nil, fmt.Errorf("delete failed: %w", err)
	}
	return map[string]interface{}{"ok": true}, nil
}

func (m *Manager) mkdir(params map[string]interface{}) (interface{}, error) {
	sessionID, _ := params["sessionId"].(string)
	path, _ := params["path"].(string)

	if sessionID == "" || path == "" {
		return nil, fmt.Errorf("sessionId and path are required")
	}

	client, err := m.getClient(sessionID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if err := client.MkdirAll(path); err != nil {
		return nil, fmt.Errorf("mkdir failed: %w", err)
	}
	return map[string]interface{}{"ok": true}, nil
}
