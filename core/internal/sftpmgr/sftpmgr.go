package sftpmgr

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/devterm/core/internal/rpc"
	"github.com/devterm/core/internal/sshmgr"
	"github.com/google/uuid"
	"github.com/pkg/sftp"
)

// Manager handles SFTP file operations with client caching.
type Manager struct {
	sshMgr     *sshmgr.Manager
	clients    map[string]*sftp.Client // sessionID -> cached client
	mu         sync.Mutex
	transferMu sync.Mutex
	transfers  map[string]chan struct{}
}

func reportProgress(notifier *rpc.Notifier, transferID, direction, fileName string, written, total int64, started time.Time, status string) {
	if notifier == nil {
		return
	}
	elapsed := time.Since(started).Seconds()
	if elapsed < 0.001 {
		elapsed = 0.001
	}
	bytesPerSec := float64(written) / elapsed
	percent := 0
	etaSeconds := 0
	if total > 0 {
		percent = int(float64(written) / float64(total) * 100)
		if bytesPerSec > 0 {
			etaSeconds = int(float64(total-written) / bytesPerSec)
		}
	}
	notifier.Notify("sftp.progress", map[string]interface{}{
		"transferId": transferID, "direction": direction, "fileName": fileName,
		"bytesTransferred": written, "totalBytes": total, "bytesPerSec": int64(bytesPerSec),
		"etaSeconds": etaSeconds, "percent": percent, "status": status,
	})
}

// New creates a new SFTP manager.
func New(sshMgr *sshmgr.Manager) *Manager {
	return &Manager{
		sshMgr:    sshMgr,
		clients:   make(map[string]*sftp.Client),
		transfers: make(map[string]chan struct{}),
	}
}

// RegisterRPC registers SFTP RPC methods.
func (m *Manager) RegisterRPC(d *rpc.Dispatcher) {
	d.Register("sftp.list", m.list)
	d.Register("sftp.upload", m.upload)
	d.Register("sftp.download", m.download)
	d.Register("sftp.rename", m.rename)
	d.Register("sftp.delete", m.deleteFile)
	d.Register("sftp.mkdir", m.mkdir)
	d.Register("sftp.compress", m.compress)
	d.Register("sftp.cancel", m.cancel)
}

func (m *Manager) beginTransfer(id string) chan struct{} {
	m.transferMu.Lock()
	defer m.transferMu.Unlock()
	stop := make(chan struct{})
	m.transfers[id] = stop
	return stop
}

func (m *Manager) endTransfer(id string) {
	m.transferMu.Lock()
	delete(m.transfers, id)
	m.transferMu.Unlock()
}

func (m *Manager) cancel(params map[string]interface{}) (interface{}, error) {
	id, _ := params["transferId"].(string)
	if id == "" {
		return nil, fmt.Errorf("transferId is required")
	}
	m.transferMu.Lock()
	stop, ok := m.transfers[id]
	if ok {
		delete(m.transfers, id)
		close(stop)
	}
	m.transferMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("transfer is no longer active")
	}
	return map[string]bool{"ok": true}, nil
}

// compress creates a ZIP archive on the remote host without invoking a shell.
func (m *Manager) compress(params map[string]interface{}) (interface{}, error) {
	sessionID, _ := params["sessionId"].(string)
	sourcePath, _ := params["path"].(string)
	if sessionID == "" || sourcePath == "" {
		return nil, fmt.Errorf("sessionId and path are required")
	}
	client, err := m.getClient(sessionID)
	if err != nil {
		return nil, err
	}
	archivePath, _ := params["archivePath"].(string)
	if archivePath == "" {
		archivePath = sourcePath + ".zip"
	}
	output, err := client.Create(archivePath)
	if err != nil {
		return nil, fmt.Errorf("creating archive: %w", err)
	}
	defer output.Close()
	writer := zip.NewWriter(output)
	defer writer.Close()
	base := pathpkg.Base(sourcePath)
	walker := client.Walk(sourcePath)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return nil, fmt.Errorf("reading source: %w", err)
		}
		info := walker.Stat()
		if info == nil {
			continue
		}
		rel := strings.TrimPrefix(walker.Path(), sourcePath)
		rel = strings.TrimPrefix(rel, "/")
		name := base
		if rel != "" {
			name += "/" + rel
		}
		if info.IsDir() {
			if _, err := writer.Create(name + "/"); err != nil {
				return nil, err
			}
			continue
		}
		input, err := client.Open(walker.Path())
		if err != nil {
			return nil, err
		}
		entry, err := writer.Create(name)
		if err == nil {
			_, err = io.Copy(entry, input)
		}
		input.Close()
		if err != nil {
			return nil, fmt.Errorf("adding %s: %w", name, err)
		}
	}
	return map[string]interface{}{"path": archivePath}, nil
}

func (m *Manager) getClient(sessionID string) (*sftp.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return cached client if available
	if client, ok := m.clients[sessionID]; ok {
		// Test if still alive
		if _, err := client.Getwd(); err == nil {
			return client, nil
		}
		// Dead client, remove from cache
		client.Close()
		delete(m.clients, sessionID)
	}

	sshClient, err := m.sshMgr.GetClient(sessionID)
	if err != nil {
		return nil, err
	}
	client, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, err
	}
	m.clients[sessionID] = client
	return client, nil
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
	transferId := uuid.New().String()
	var stop chan struct{}
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
	stop = m.beginTransfer(transferId)

	// Copy in chunks with progress
	go func() {
		defer m.endTransfer(transferId)
		defer localFile.Close()
		defer remoteFile.Close()

		buf := make([]byte, 32*1024)
		var written int64
		started := time.Now()
		lastProgress := started.Add(-200 * time.Millisecond)

		for {
			select {
			case <-stop:
				reportProgress(notifier, transferId, "upload", filepath.Base(localPath), written, totalSize, started, "cancelled")
				if notifier != nil {
					notifier.Notify("sftp.complete", map[string]interface{}{"transferId": transferId, "status": "cancelled"})
				}
				return
			default:
			}
			n, readErr := localFile.Read(buf)
			if n > 0 {
				_, writeErr := remoteFile.Write(buf[:n])
				if writeErr != nil {
					reportProgress(notifier, transferId, "upload", filepath.Base(localPath), written, totalSize, started, "failed")
					return
				}
				written += int64(n)
				if time.Since(lastProgress) >= 120*time.Millisecond || written == totalSize {
					reportProgress(notifier, transferId, "upload", filepath.Base(localPath), written, totalSize, started, "in_progress")
					lastProgress = time.Now()
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				reportProgress(notifier, transferId, "upload", filepath.Base(localPath), written, totalSize, started, "failed")
				return
			}
		}

		reportProgress(notifier, transferId, "upload", filepath.Base(localPath), written, totalSize, started, "complete")
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
	// A local directory is a convenient target for drag-and-drop downloads.
	if info, statErr := os.Stat(localPath); statErr == nil && info.IsDir() {
		localPath = filepath.Join(localPath, filepath.Base(remotePath))
	}

	client, err := m.getClient(sessionID)
	if err != nil {
		return nil, err
	}
	transferId := uuid.New().String()
	var stop chan struct{}
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
	stop = m.beginTransfer(transferId)

	// Copy in chunks with progress
	go func() {
		defer m.endTransfer(transferId)
		defer remoteFile.Close()
		defer localFile.Close()

		buf := make([]byte, 32*1024)
		var written int64
		started := time.Now()
		lastProgress := started.Add(-200 * time.Millisecond)

		for {
			select {
			case <-stop:
				reportProgress(notifier, transferId, "download", filepath.Base(remotePath), written, totalSize, started, "cancelled")
				if notifier != nil {
					notifier.Notify("sftp.complete", map[string]interface{}{"transferId": transferId, "status": "cancelled"})
				}
				return
			default:
			}
			n, readErr := remoteFile.Read(buf)
			if n > 0 {
				_, writeErr := localFile.Write(buf[:n])
				if writeErr != nil {
					reportProgress(notifier, transferId, "download", filepath.Base(remotePath), written, totalSize, started, "failed")
					return
				}
				written += int64(n)
				if time.Since(lastProgress) >= 120*time.Millisecond || written == totalSize {
					reportProgress(notifier, transferId, "download", filepath.Base(remotePath), written, totalSize, started, "in_progress")
					lastProgress = time.Now()
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				reportProgress(notifier, transferId, "download", filepath.Base(remotePath), written, totalSize, started, "failed")
				return
			}
		}

		reportProgress(notifier, transferId, "download", filepath.Base(remotePath), written, totalSize, started, "complete")
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
	if err := client.MkdirAll(path); err != nil {
		return nil, fmt.Errorf("mkdir failed: %w", err)
	}
	return map[string]interface{}{"ok": true}, nil
}
