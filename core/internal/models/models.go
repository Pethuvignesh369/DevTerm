package models

// Host represents a saved SSH host.
type Host struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Hostname   string   `json:"hostname"`
	Port       int      `json:"port"`
	Username   string   `json:"username"`
	IdentityID *string  `json:"identityId,omitempty"`
	GroupID    *int     `json:"groupId,omitempty"`
	Favorite   bool     `json:"favorite"`
	Tags       []string `json:"tags"`
	CreatedAt  string   `json:"createdAt"`
	UpdatedAt  string   `json:"updatedAt"`
}

// Group represents a host group.
type Group struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

// Identity represents an authentication identity.
type Identity struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	AuthType  string  `json:"authType"` // "password", "key", "agent"
	SSHKeyID  *string `json:"sshKeyId,omitempty"`
	VaultRef  *string `json:"vaultRef,omitempty"`
	CreatedAt string  `json:"createdAt"`
}

// SSHKey represents stored SSH key metadata.
type SSHKey struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	KeyType             string `json:"keyType"` // "rsa", "ed25519"
	PublicKey           string `json:"publicKey"`
	Fingerprint         string `json:"fingerprint"`
	PassphraseProtected bool   `json:"passphraseProtected"`
	VaultRef            string `json:"vaultRef"`
	CreatedAt           string `json:"createdAt"`
}

// Snippet represents a saved command snippet.
type Snippet struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Command   string   `json:"command"`
	Tags      []string `json:"tags"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

// HistoryEntry represents a command history record.
type HistoryEntry struct {
	ID         int    `json:"id"`
	HostID     string `json:"hostId"`
	Command    string `json:"command"`
	ExecutedAt string `json:"executedAt"`
}

// PortForward represents a port forwarding rule.
type PortForward struct {
	ID         string  `json:"id"`
	HostID     string  `json:"hostId"`
	Type       string  `json:"type"` // "local", "remote", "dynamic"
	LocalHost  string  `json:"localHost"`
	LocalPort  int     `json:"localPort"`
	RemoteHost *string `json:"remoteHost,omitempty"`
	RemotePort *int    `json:"remotePort,omitempty"`
	CreatedAt  string  `json:"createdAt"`
}

// Metrics represents system metrics for a remote host.
type Metrics struct {
	CPU        *float64          `json:"cpu,omitempty"`
	MemTotal   *uint64           `json:"memTotal,omitempty"`
	MemUsed    *uint64           `json:"memUsed,omitempty"`
	DiskTotal  *uint64           `json:"diskTotal,omitempty"`
	DiskUsed   *uint64           `json:"diskUsed,omitempty"`
	NetRxBytes *uint64           `json:"netRxBytes,omitempty"`
	NetTxBytes *uint64           `json:"netTxBytes,omitempty"`
	Uptime     *string           `json:"uptime,omitempty"`
	Unavailable map[string]bool  `json:"unavailable,omitempty"`
}
