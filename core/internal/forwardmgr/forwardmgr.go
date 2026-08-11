package forwardmgr

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/devterm/core/internal/rpc"
	"github.com/devterm/core/internal/sshmgr"
	"github.com/google/uuid"
)

// Manager handles port forwarding (local, remote, dynamic SOCKS).
type Manager struct {
	db     *sql.DB
	sshMgr *sshmgr.Manager
	mu     sync.Mutex
	active map[string]chan struct{} // id -> stop channel
}

// New creates a new port forward manager.
func New(db *sql.DB, sshMgr *sshmgr.Manager) *Manager {
	return &Manager{
		db:     db,
		sshMgr: sshMgr,
		active: make(map[string]chan struct{}),
	}
}

// RegisterRPC registers port forwarding RPC methods.
func (m *Manager) RegisterRPC(d *rpc.Dispatcher) {
	d.Register("forward.start", m.start)
	d.Register("forward.stop", m.stop)
	d.Register("forward.list", m.list)
}

func (m *Manager) start(params map[string]interface{}) (interface{}, error) {
	sessionID, _ := params["sessionId"].(string)
	fwdType, _ := params["type"].(string)
	localHost, _ := params["localHost"].(string)
	localPort, _ := params["localPort"].(float64)
	remoteHost, _ := params["remoteHost"].(string)
	remotePort, _ := params["remotePort"].(float64)

	if sessionID == "" || fwdType == "" {
		return nil, fmt.Errorf("sessionId and type are required")
	}
	if localHost == "" {
		localHost = "127.0.0.1"
	}

	client, err := m.sshMgr.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	id := uuid.New().String()
	stopCh := make(chan struct{})

	sess, _ := m.sshMgr.GetSession(sessionID)
	hostID := ""
	if sess != nil {
		hostID = sess.HostID
	}

	switch fwdType {
	case "local":
		if remoteHost == "" || remotePort == 0 {
			return nil, fmt.Errorf("remoteHost and remotePort are required for local forwarding")
		}
		localAddr := net.JoinHostPort(localHost, fmt.Sprintf("%d", int(localPort)))
		remoteAddr := net.JoinHostPort(remoteHost, fmt.Sprintf("%d", int(remotePort)))

		listener, err := net.Listen("tcp", localAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to listen on %s: %w", localAddr, err)
		}

		go func() {
			defer listener.Close()
			for {
				select {
				case <-stopCh:
					return
				default:
				}
				conn, err := listener.Accept()
				if err != nil {
					select {
					case <-stopCh:
						return
					default:
						log.Printf("[forward] accept error: %v", err)
						continue
					}
				}
				go func() {
					remote, err := client.Dial("tcp", remoteAddr)
					if err != nil {
						conn.Close()
						log.Printf("[forward] dial remote %s failed: %v", remoteAddr, err)
						return
					}
					go io.Copy(remote, conn)
					io.Copy(conn, remote)
					conn.Close()
					remote.Close()
				}()
			}
		}()

	case "remote":
		if remoteHost == "" || remotePort == 0 {
			return nil, fmt.Errorf("remoteHost and remotePort are required for remote forwarding")
		}
		remoteAddr := net.JoinHostPort(remoteHost, fmt.Sprintf("%d", int(remotePort)))
		localAddr := net.JoinHostPort(localHost, fmt.Sprintf("%d", int(localPort)))

		listener, err := client.Listen("tcp", remoteAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to listen on remote %s: %w", remoteAddr, err)
		}

		go func() {
			defer listener.Close()
			for {
				select {
				case <-stopCh:
					return
				default:
				}
				conn, err := listener.Accept()
				if err != nil {
					select {
					case <-stopCh:
						return
					default:
						log.Printf("[forward] remote accept error: %v", err)
						continue
					}
				}
				go func() {
					local, err := net.Dial("tcp", localAddr)
					if err != nil {
						conn.Close()
						log.Printf("[forward] dial local %s failed: %v", localAddr, err)
						return
					}
					go io.Copy(local, conn)
					io.Copy(conn, local)
					conn.Close()
					local.Close()
				}()
			}
		}()

	case "dynamic":
		// SOCKS5 proxy - simplified implementation
		localAddr := net.JoinHostPort(localHost, fmt.Sprintf("%d", int(localPort)))
		listener, err := net.Listen("tcp", localAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to listen on %s: %w", localAddr, err)
		}

		go func() {
			defer listener.Close()
			for {
				select {
				case <-stopCh:
					return
				default:
				}
				conn, err := listener.Accept()
				if err != nil {
					select {
					case <-stopCh:
						return
					default:
						continue
					}
				}
				go handleSocks5(conn, client, stopCh)
			}
		}()

	default:
		return nil, fmt.Errorf("unsupported forward type: %s", fwdType)
	}

	m.mu.Lock()
	m.active[id] = stopCh
	m.mu.Unlock()

	// Persist the rule
	var rh, rp interface{}
	if remoteHost != "" {
		rh = remoteHost
	}
	if remotePort != 0 {
		rp = int(remotePort)
	}
	m.db.Exec(
		`INSERT INTO port_forwards (id, host_id, type, local_host, local_port, remote_host, remote_port) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, hostID, fwdType, localHost, int(localPort), rh, rp,
	)

	return map[string]interface{}{"id": id, "status": "active"}, nil
}

func (m *Manager) stop(params map[string]interface{}) (interface{}, error) {
	id, _ := params["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	m.mu.Lock()
	stopCh, ok := m.active[id]
	if ok {
		close(stopCh)
		delete(m.active, id)
	}
	m.mu.Unlock()

	m.db.Exec(`DELETE FROM port_forwards WHERE id = ?`, id)

	if !ok {
		return nil, fmt.Errorf("forward not found or not active")
	}
	return map[string]interface{}{"ok": true}, nil
}

func (m *Manager) list(params map[string]interface{}) (interface{}, error) {
	rows, err := m.db.Query(`SELECT id, host_id, type, local_host, local_port, remote_host, remote_port, created_at FROM port_forwards`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forwards []map[string]interface{}
	for rows.Next() {
		var id, hostID, fwdType, localHost, createdAt string
		var localPort int
		var remoteHost sql.NullString
		var remotePort sql.NullInt64
		if err := rows.Scan(&id, &hostID, &fwdType, &localHost, &localPort, &remoteHost, &remotePort, &createdAt); err != nil {
			return nil, err
		}
		entry := map[string]interface{}{
			"id":        id,
			"hostId":    hostID,
			"type":      fwdType,
			"localHost": localHost,
			"localPort": localPort,
			"createdAt": createdAt,
			"active":    m.isActive(id),
		}
		if remoteHost.Valid {
			entry["remoteHost"] = remoteHost.String
		}
		if remotePort.Valid {
			entry["remotePort"] = int(remotePort.Int64)
		}
		forwards = append(forwards, entry)
	}
	if forwards == nil {
		forwards = []map[string]interface{}{}
	}
	return forwards, nil
}

func (m *Manager) isActive(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.active[id]
	return ok
}

// handleSocks5 handles a minimal SOCKS5 connection
func handleSocks5(conn net.Conn, sshClient interface {
	Dial(string, string) (net.Conn, error)
}, stopCh chan struct{}) {
	defer conn.Close()

	// SOCKS5 handshake
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil || n < 2 || buf[0] != 0x05 {
		return
	}
	// No auth required
	conn.Write([]byte{0x05, 0x00})

	// Read request
	n, err = conn.Read(buf)
	if err != nil || n < 7 {
		return
	}
	if buf[1] != 0x01 { // Only CONNECT supported
		conn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	var targetAddr string
	switch buf[3] {
	case 0x01: // IPv4
		if n < 10 {
			return
		}
		targetAddr = fmt.Sprintf("%d.%d.%d.%d:%d", buf[4], buf[5], buf[6], buf[7], int(buf[8])<<8|int(buf[9]))
	case 0x03: // Domain
		domainLen := int(buf[4])
		if n < 5+domainLen+2 {
			return
		}
		domain := string(buf[5 : 5+domainLen])
		port := int(buf[5+domainLen])<<8 | int(buf[5+domainLen+1])
		targetAddr = fmt.Sprintf("%s:%d", domain, port)
	case 0x04: // IPv6
		if n < 22 {
			return
		}
		// Skip IPv6 for now
		conn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	default:
		return
	}

	// Dial through SSH
	remote, err := sshClient.Dial("tcp", targetAddr)
	if err != nil {
		conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	// Success response
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	// Proxy data
	go io.Copy(remote, conn)
	io.Copy(conn, remote)
	remote.Close()
}
