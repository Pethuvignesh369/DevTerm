package hostmgr

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// SSHConfigEntry represents a parsed SSH config host entry.
type SSHConfigEntry struct {
	Name         string `json:"name"`
	Hostname     string `json:"hostname"`
	Port         int    `json:"port"`
	Username     string `json:"username"`
	IdentityFile string `json:"identityFile,omitempty"`
}

// ParseSSHConfig reads and parses the default SSH config file.
func ParseSSHConfig() ([]SSHConfigEntry, error) {
	configPath := defaultSSHConfigPath()
	return ParseSSHConfigFile(configPath)
}

// ParseSSHConfigFile parses a specific SSH config file.
func ParseSSHConfigFile(path string) ([]SSHConfigEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read SSH config: %w", err)
	}
	defer file.Close()

	var entries []SSHConfigEntry
	var current *SSHConfigEntry

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split key value
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			parts = strings.SplitN(line, "\t", 2)
			if len(parts) < 2 {
				continue
			}
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch strings.ToLower(key) {
		case "host":
			// Skip wildcard entries
			if strings.Contains(value, "*") || strings.Contains(value, "?") {
				current = nil
				continue
			}
			// Start new entry
			entry := SSHConfigEntry{
				Name: value,
				Port: 22,
			}
			current = &entry
			entries = append(entries, entry)

		case "hostname":
			if current != nil {
				entries[len(entries)-1].Hostname = value
			}

		case "port":
			if current != nil {
				if p, err := strconv.Atoi(value); err == nil {
					entries[len(entries)-1].Port = p
				}
			}

		case "user":
			if current != nil {
				entries[len(entries)-1].Username = value
			}

		case "identityfile":
			if current != nil {
				// Expand ~ to home dir
				if strings.HasPrefix(value, "~/") {
					home, _ := os.UserHomeDir()
					value = filepath.Join(home, value[2:])
				}
				entries[len(entries)-1].IdentityFile = value
			}
		}
	}

	// Filter out entries without hostname
	var valid []SSHConfigEntry
	for _, e := range entries {
		if e.Hostname != "" {
			if e.Username == "" {
				e.Username = currentUsername()
			}
			valid = append(valid, e)
		}
	}

	return valid, nil
}

func defaultSSHConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ssh", "config")
}

func currentUsername() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("USERNAME")
	}
	return os.Getenv("USER")
}
