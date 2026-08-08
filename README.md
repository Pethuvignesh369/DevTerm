# DevTerm

<p align="center">
  <img src="https://img.shields.io/badge/version-0.1.0-blue" alt="Version" />
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey" alt="Platform" />
  <img src="https://img.shields.io/badge/license-MIT-green" alt="License" />
</p>

<p align="center">
  A modern, cross-platform SSH client built for DevOps engineers, SREs, and Platform teams.
</p>

---

## Features

- **SSH Terminal** - Multi-tab terminal with split panes, search, themes
- **Host Management** - Save, organize, tag, and search your servers
- **SSH Key Management** - Generate (RSA/ED25519) and import keys securely
- **SFTP File Browser** - Upload, download, rename, delete with progress
- **Port Forwarding** - Local, remote, and dynamic SOCKS tunnels
- **System Dashboard** - Real-time CPU, memory, disk, network monitoring
- **Command Snippets** - Save and reuse frequently used commands
- **Command History** - Searchable per-host command history
- **Secure Vault** - AES-256-GCM encrypted secrets, OS keychain integration
- **Zero Telemetry** - No analytics, no tracking, fully offline-capable

## Architecture

```
Vue 3 UI (TypeScript + Tailwind + shadcn)
         |
    Tauri IPC (Rust)
         |
  Go Sidecar (JSON-RPC over stdio)
   ├── SSH Manager
   ├── SFTP Manager
   ├── Key Manager
   ├── Host Manager
   ├── Port Forward Manager
   ├── Monitor Manager
   ├── History Manager
   ├── Snippet Manager
   └── Secret Vault
         |
      SQLite (local)
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Desktop | [Tauri v2](https://v2.tauri.app/) (Rust) |
| Frontend | Vue 3, TypeScript, Tailwind CSS, Pinia |
| Terminal | xterm.js |
| Backend | Go 1.22+ |
| Database | SQLite (modernc.org/sqlite, pure Go) |
| SSH | golang.org/x/crypto/ssh |
| SFTP | pkg/sftp |
| Security | OS Keychain + AES-256-GCM fallback |

## Getting Started

### Prerequisites

- [Node.js](https://nodejs.org/) 18+
- [Rust](https://rustup.rs/) (with GNU or MSVC toolchain)
- [Go](https://go.dev/) 1.22+
- [MSYS2](https://www.msys2.org/) (Windows, for GNU toolchain)

### Setup

```bash
# Clone the repository
git clone https://github.com/Pethuvignesh/DevTerm.git
cd DevTerm

# Install frontend dependencies
npm install

# Build the Go sidecar
cd core
go mod tidy
# Windows:
powershell ./build.ps1
# Or manually:
set CGO_ENABLED=0 && go build -o ../src-tauri/binaries/devterm-core-x86_64-pc-windows-gnu.exe ./cmd/devterm-core/
cd ..

# Run in development mode
# (Ensure C:\msys64\mingw64\bin is in PATH for Windows GNU toolchain)
npm run tauri dev
```

### Build for Production

```bash
npm run tauri build
```

## Project Structure

```
DevTerm/
├── src/                    # Vue 3 frontend
│   ├── components/         # UI components
│   ├── views/              # Page views
│   ├── stores/             # Pinia state management
│   ├── composables/        # Vue composables
│   └── lib/                # Utilities (RPC client, errors)
├── src-tauri/              # Rust Tauri shell
│   ├── src/                # Rust source (IPC broker)
│   └── binaries/           # Go sidecar binaries
├── core/                   # Go sidecar
│   ├── cmd/devterm-core/   # Entry point
│   └── internal/           # Business logic
│       ├── rpc/            # JSON-RPC dispatcher
│       ├── sshmgr/         # SSH session management
│       ├── hostmgr/        # Host CRUD
│       ├── keymgr/         # Key generation/import
│       ├── sftpmgr/        # SFTP operations
│       ├── forwardmgr/     # Port forwarding
│       ├── historymgr/     # Command history
│       ├── snippetmgr/     # Command snippets
│       ├── monitor/        # System metrics
│       ├── vault/          # Secret storage
│       ├── db/             # SQLite + migrations
│       └── models/         # Data models
└── .kiro/                  # Spec files
```

## Security

- Private keys and passwords are never stored in plaintext
- Secrets use OS keychain (Windows Credential Manager / macOS Keychain / Linux Secret Service)
- Fallback: AES-256-GCM encrypted file vault
- No telemetry or analytics - zero outbound network calls except to your configured hosts
- IPC uses stdio pipes (no local network ports opened)

## Roadmap

- [x] v1.0 - SSH, SFTP, Host Management, Terminal, Port Forwarding
- [ ] v2.0 - Docker, Kubernetes, Monitoring, AI Assistant
- [ ] v3.0 - AWS, Azure, GCP, Session Recording
- [ ] v4.0 - Plugin Marketplace, Team Collaboration

## License

MIT
