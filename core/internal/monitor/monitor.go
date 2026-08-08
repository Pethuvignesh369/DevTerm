package monitor

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/devterm/core/internal/rpc"
	"github.com/devterm/core/internal/sshmgr"
)

// Manager handles remote system monitoring via SSH commands.
type Manager struct {
	sshMgr  *sshmgr.Manager
	mu      sync.Mutex
	running map[string]chan struct{} // sessionID -> stop channel
}

// New creates a new monitor manager.
func New(sshMgr *sshmgr.Manager) *Manager {
	return &Manager{
		sshMgr:  sshMgr,
		running: make(map[string]chan struct{}),
	}
}

// RegisterRPC registers monitoring RPC methods.
func (m *Manager) RegisterRPC(d *rpc.Dispatcher) {
	d.Register("monitor.start", m.start)
	d.Register("monitor.stop", m.stop)
}

func (m *Manager) start(params map[string]interface{}) (interface{}, error) {
	sessionID, _ := params["sessionId"].(string)
	if sessionID == "" {
		return nil, fmt.Errorf("sessionId is required")
	}

	interval := 3000 // ms
	if i, ok := params["interval"].(float64); ok {
		interval = int(i)
	}

	// Check session exists
	_, err := m.sshMgr.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	if _, running := m.running[sessionID]; running {
		m.mu.Unlock()
		return map[string]interface{}{"status": "already_running"}, nil
	}
	stopCh := make(chan struct{})
	m.running[sessionID] = stopCh
	m.mu.Unlock()

	// Start polling
	go m.poll(sessionID, time.Duration(interval)*time.Millisecond, stopCh)

	return map[string]interface{}{"status": "started"}, nil
}

func (m *Manager) stop(params map[string]interface{}) (interface{}, error) {
	sessionID, _ := params["sessionId"].(string)
	if sessionID == "" {
		return nil, fmt.Errorf("sessionId is required")
	}

	m.mu.Lock()
	stopCh, ok := m.running[sessionID]
	if ok {
		close(stopCh)
		delete(m.running, sessionID)
	}
	m.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("monitoring not running for session")
	}
	return map[string]interface{}{"ok": true}, nil
}

func (m *Manager) poll(sessionID string, interval time.Duration, stopCh chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	notifier := rpc.GetNotifier()
	var prevIdle, prevTotal uint64
	first := true

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			metrics := m.collectMetrics(sessionID, &prevIdle, &prevTotal, &first)
			if notifier != nil {
				notifier.Notify("monitor.tick", map[string]interface{}{
					"sessionId": sessionID,
					"metrics":   metrics,
				})
			}
		}
	}
}

func (m *Manager) runCommand(sessionID, cmd string) (string, error) {
	client, err := m.sshMgr.GetClient(sessionID)
	if err != nil {
		return "", err
	}
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (m *Manager) collectMetrics(sessionID string, prevIdle, prevTotal *uint64, first *bool) map[string]interface{} {
	metrics := map[string]interface{}{}
	unavailable := map[string]bool{}

	// CPU from /proc/stat
	cpuOutput, err := m.runCommand(sessionID, "head -1 /proc/stat")
	if err == nil {
		fields := strings.Fields(cpuOutput)
		if len(fields) >= 5 && fields[0] == "cpu" {
			var total, idle uint64
			for i := 1; i < len(fields); i++ {
				val, _ := strconv.ParseUint(fields[i], 10, 64)
				total += val
				if i == 4 { // idle is the 4th value (index 4)
					idle = val
				}
			}
			if !*first {
				totalDelta := total - *prevTotal
				idleDelta := idle - *prevIdle
				if totalDelta > 0 {
					cpuPercent := float64(totalDelta-idleDelta) / float64(totalDelta) * 100
					metrics["cpu"] = int(cpuPercent)
				}
			}
			*prevTotal = total
			*prevIdle = idle
			*first = false
		}
	} else {
		unavailable["cpu"] = true
	}

	// Memory from /proc/meminfo
	memOutput, err := m.runCommand(sessionID, "head -4 /proc/meminfo")
	if err == nil {
		var memTotal, memAvailable, memFree uint64
		for _, line := range strings.Split(memOutput, "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseUint(fields[1], 10, 64)
				switch {
				case strings.HasPrefix(line, "MemTotal:"):
					memTotal = val
				case strings.HasPrefix(line, "MemFree:"):
					memFree = val
				case strings.HasPrefix(line, "MemAvailable:"):
					memAvailable = val
				}
			}
		}
		if memTotal > 0 {
			used := memTotal - memAvailable
			if memAvailable == 0 {
				used = memTotal - memFree
			}
			metrics["memTotal"] = memTotal / 1024 // MB
			metrics["memUsed"] = used / 1024       // MB
		}
	} else {
		unavailable["memory"] = true
	}

	// Disk from df
	diskOutput, err := m.runCommand(sessionID, "df -P / | tail -1")
	if err == nil {
		fields := strings.Fields(diskOutput)
		if len(fields) >= 4 {
			total, _ := strconv.ParseUint(fields[1], 10, 64)
			used, _ := strconv.ParseUint(fields[2], 10, 64)
			metrics["diskTotal"] = total / 1024 / 1024 // GB
			metrics["diskUsed"] = used / 1024 / 1024   // GB
			if total > 0 {
				metrics["diskPercent"] = int(float64(used) / float64(total) * 100)
			}
		}
	} else {
		unavailable["disk"] = true
	}

	// Network from /proc/net/dev
	netOutput, err := m.runCommand(sessionID, "cat /proc/net/dev | grep -v lo | tail -n +3 | head -1")
	if err == nil {
		fields := strings.Fields(netOutput)
		if len(fields) >= 10 {
			rxBytes, _ := strconv.ParseUint(fields[1], 10, 64)
			txBytes, _ := strconv.ParseUint(fields[9], 10, 64)
			metrics["netRx"] = rxBytes
			metrics["netTx"] = txBytes
		}
	} else {
		unavailable["network"] = true
	}

	// Uptime
	uptimeOutput, err := m.runCommand(sessionID, "cat /proc/uptime")
	if err == nil {
		fields := strings.Fields(uptimeOutput)
		if len(fields) >= 1 {
			seconds, _ := strconv.ParseFloat(fields[0], 64)
			days := int(seconds) / 86400
			hours := (int(seconds) % 86400) / 3600
			mins := (int(seconds) % 3600) / 60
			metrics["uptime"] = fmt.Sprintf("%dd %dh %dm", days, hours, mins)
		}
	} else {
		unavailable["uptime"] = true
	}

	if len(unavailable) > 0 {
		metrics["unavailable"] = unavailable
	}

	return metrics
}
