package sshmgr

import (
	"database/sql"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"github.com/devterm/core/internal/hostmgr"
	"github.com/devterm/core/internal/rpc"
	"github.com/devterm/core/internal/vault"
	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Session represents an active SSH session.
type Session struct {
	ID         string
	HostID     string
	Client     *ssh.Client
	SSHSession *ssh.Session
	Stdin      io.WriteCloser
}

// Manager manages SSH connections and sessions.
type Manager struct {
	db         *sql.DB
	vault      vault.Vault
	hostMgr    *hostmgr.Manager
	knownHosts *KnownHostsDB
	mu         sync.RWMutex
	sessions   map[string]*Session
}

// New creates a new SSH manager.
func New(db *sql.DB, v vault.Vault, hostMgr *hostmgr.Manager) *Manager {
	return &Manager{
		db:         db,
		vault:      v,
		hostMgr:    hostMgr,
		knownHosts: NewKnownHostsDB(db),
		sessions:   make(map[string]*Session),
	}
}

// RegisterRPC registers SSH session RPC methods.
func (m *Manager) RegisterRPC(d *rpc.Dispatcher) {
	d.Register("ssh.connect", m.connect)
	d.Register("ssh.disconnect", m.disconnect)
	d.Register("ssh.write", m.write)
	d.Register("ssh.resize", m.resize)
	d.Register("ssh.knownHosts.list", m.listKnownHosts)
	d.Register("ssh.knownHosts.remove", m.removeKnownHost)
}

// GetSession returns an active session by ID (used by other managers).
func (m *Manager) GetSession(sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return sess, nil
}

// GetClient returns the SSH client for a session (used by SFTP/forward/monitor).
func (m *Manager) GetClient(sessionID string) (*ssh.Client, error) {
	sess, err := m.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	return sess.Client, nil
}

func (m *Manager) connect(params map[string]interface{}) (interface{}, error) {
	hostID, _ := params["hostId"].(string)
	if hostID == "" {
		return nil, fmt.Errorf("hostId is required")
	}

	host, err := m.hostMgr.GetHost(hostID)
	if err != nil {
		return nil, fmt.Errorf("loading host: %w", err)
	}

	// Build SSH config
	config := &ssh.ClientConfig{
		User:            host.Username,
		HostKeyCallback: m.knownHosts.HostKeyCallback(),
	}

	// Resolve auth method from identity
	if host.IdentityID != nil {
		identity, err := m.hostMgr.GetIdentity(*host.IdentityID)
		if err == nil {
			switch identity.AuthType {
			case "password":
				if identity.VaultRef != nil {
					secret, err := m.vault.Get(*identity.VaultRef)
					if err == nil {
						config.Auth = append(config.Auth, ssh.Password(string(secret)))
					}
				}
			case "key":
				if identity.SSHKeyID != nil {
					// Load key from vault via key manager
					var vaultRef string
					m.db.QueryRow(`SELECT vault_ref FROM ssh_keys WHERE id = ?`, *identity.SSHKeyID).Scan(&vaultRef)
					if vaultRef != "" {
						keyData, err := m.vault.Get(vaultRef)
						if err == nil {
							signer, err := ssh.ParsePrivateKey(keyData)
							if err == nil {
								config.Auth = append(config.Auth, ssh.PublicKeys(signer))
							}
						}
					}
				}
			case "agent":
				// Connect to SSH agent
				agentSock := os.Getenv("SSH_AUTH_SOCK")
				if agentSock != "" {
					agentConn, agentErr := net.Dial("unix", agentSock)
					if agentErr == nil {
						agentClient := agent.NewClient(agentConn)
						config.Auth = append(config.Auth, ssh.PublicKeysCallback(agentClient.Signers))
					}
				}
			}
		}
	}

	// Connect
	addr := fmt.Sprintf("%s:%d", host.Hostname, host.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}

	sessionID := uuid.New().String()
	sess := &Session{
		ID:     sessionID,
		HostID: hostID,
		Client: client,
	}

	// Open a session and request a PTY
	sshSession, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("opening session: %w", err)
	}

	// Request PTY
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	cols := 80
	rows := 24
	if c, ok := params["cols"].(float64); ok {
		cols = int(c)
	}
	if r, ok := params["rows"].(float64); ok {
		rows = int(r)
	}
	if err := sshSession.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("requesting PTY: %w", err)
	}

	// Get I/O pipes
	stdin, err := sshSession.StdinPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return nil, err
	}
	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return nil, err
	}

	// Start shell
	if err := sshSession.Shell(); err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("starting shell: %w", err)
	}

	// Store stdin writer and session for ssh.write and ssh.resize
	sess.Stdin = stdin
	sess.SSHSession = sshSession

	m.mu.Lock()
	m.sessions[sessionID] = sess
	m.mu.Unlock()

	// Record session in DB
	m.db.Exec(`INSERT INTO sessions (id, host_id) VALUES (?, ?)`, sessionID, hostID)

	// Start reading stdout and pushing notifications
	notifier := rpc.GetNotifier()
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 && notifier != nil {
				notifier.Notify("terminal.data", map[string]interface{}{
					"sessionId": sessionID,
					"chunk":     string(buf[:n]),
				})
			}
			if err != nil {
				break
			}
		}
		// Session ended
		m.mu.Lock()
		delete(m.sessions, sessionID)
		m.mu.Unlock()
		m.db.Exec(`UPDATE sessions SET ended_at = datetime('now') WHERE id = ?`, sessionID)
		if notifier != nil {
			notifier.Notify("ssh.status", map[string]interface{}{
				"sessionId": sessionID,
				"status":    "disconnected",
			})
		}
	}()

	return map[string]interface{}{
		"sessionId": sessionID,
		"hostId":    hostID,
	}, nil
}

func (m *Manager) disconnect(params map[string]interface{}) (interface{}, error) {
	sessionID, _ := params["sessionId"].(string)
	m.mu.Lock()
	sess, ok := m.sessions[sessionID]
	if ok {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	sess.Client.Close()
	m.db.Exec(`UPDATE sessions SET ended_at = datetime('now') WHERE id = ?`, sessionID)
	return map[string]interface{}{"ok": true}, nil
}

func (m *Manager) write(params map[string]interface{}) (interface{}, error) {
	sessionID, _ := params["sessionId"].(string)
	data, _ := params["data"].(string)
	if sessionID == "" || data == "" {
		return nil, fmt.Errorf("sessionId and data are required")
	}

	m.mu.RLock()
	sess, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	if sess.Stdin == nil {
		return nil, fmt.Errorf("session stdin not available")
	}

	_, err := sess.Stdin.Write([]byte(data))
	if err != nil {
		return nil, fmt.Errorf("write failed: %w", err)
	}
	return map[string]interface{}{"ok": true}, nil
}

func (m *Manager) resize(params map[string]interface{}) (interface{}, error) {
	sessionID, _ := params["sessionId"].(string)
	cols, _ := params["cols"].(float64)
	rows, _ := params["rows"].(float64)
	if sessionID == "" {
		return nil, fmt.Errorf("sessionId is required")
	}

	m.mu.RLock()
	sess, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	if sess.SSHSession == nil {
		return nil, fmt.Errorf("session not available for resize")
	}

	err := sess.SSHSession.WindowChange(int(rows), int(cols))
	if err != nil {
		return nil, fmt.Errorf("resize failed: %w", err)
	}
	return map[string]interface{}{"ok": true}, nil
}


func (m *Manager) listKnownHosts(params map[string]interface{}) (interface{}, error) {
	return m.knownHosts.ListHosts()
}

func (m *Manager) removeKnownHost(params map[string]interface{}) (interface{}, error) {
	hostname, _ := params["hostname"].(string)
	port, _ := params["port"].(string)
	if hostname == "" {
		return nil, fmt.Errorf("hostname is required")
	}
	if port == "" {
		port = "22"
	}
	err := m.knownHosts.RemoveHost(hostname, port)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"ok": true}, nil
}
