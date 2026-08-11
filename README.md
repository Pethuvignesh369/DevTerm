# DevTerm

<p align="center">
  <img src="https://img.shields.io/badge/version-0.1.0-blue" alt="Version" />
  <img src="https://img.shields.io/badge/desktop-Tauri%202-24C8DB" alt="Tauri 2" />
  <img src="https://img.shields.io/badge/license-MIT-green" alt="MIT License" />
</p>

<p align="center">
  A local-first desktop SSH workspace for engineers who need terminals, hosts, files, tunnels, and operational context in one place.
</p>

## Highlights

- Multi-tab SSH terminal with split panes, search, link detection, clipboard actions, zoom, and configurable themes.
- Host, identity, group, and SSH-key management—including RSA and Ed25519 generation and passphrase-protected keys.
- SFTP browser for navigation, rename, delete, folder creation, and downloading remote files with progress events.
- Local, remote, and SOCKS port forwarding; command snippets; searchable history; and a Linux remote-metrics dashboard.
- Command palette and keyboard-first navigation for common actions.
- Local SQLite persistence, trust-on-first-use host-key verification, and OS-keychain secret storage with an AES-256-GCM fallback.
- No telemetry and no background network activity beyond the hosts you configure.

## Architecture

```text
Vue 3 desktop UI (TypeScript, Pinia, Tailwind, xterm.js)
                         │
                  Tauri commands/events
                         │
              Rust JSON-RPC sidecar bridge
                         │
          Go service layer over stdio JSON-RPC
    ├── SSH / SFTP / port forwarding
    ├── hosts / keys / history / snippets / monitoring
    ├── secure vault and known-hosts verification
    └── SQLite database and settings
```

## Stack

| Layer | Technology |
| --- | --- |
| Desktop shell | Tauri v2 and Rust |
| UI | Vue 3, TypeScript, Pinia, Tailwind CSS |
| Terminal | xterm.js |
| Core services | Go 1.22+ |
| Persistence | SQLite (`modernc.org/sqlite`) |
| SSH / files | `golang.org/x/crypto/ssh`, `pkg/sftp` |

## Getting started

### Prerequisites

- Node.js 18+
- Go 1.22+
- Rust toolchain compatible with Tauri
- On Windows GNU builds, MSYS2 with `C:\msys64\mingw64\bin` on `PATH`

### Run locally

```bash
git clone https://github.com/Pethuvignesh369/DevTerm.git
cd DevTerm
npm install

# Build the Go sidecar (Windows)
cd core
powershell ./build.ps1
cd ..

npm run tauri dev
```

### Validate and package

```bash
# Frontend type-check and production bundle
npm run build

# Go compilation checks
cd core && go test ./...

# Desktop bundle
cd .. && npm run tauri build
```

## Project layout

```text
src/                 Vue views, components, stores, and composables
src-tauri/           Tauri shell and Go sidecar bridge
core/cmd/            Go sidecar entry point
core/internal/       SSH, SFTP, vault, persistence, and RPC services
```

## Security model

- Passwords and private keys are kept out of SQLite and stored in the OS keychain when available.
- The fallback vault encrypts secret data with AES-256-GCM.
- SSH host keys are trusted on first connection, then checked on subsequent connections; changed keys are rejected.
- App settings, hosts, known hosts, history, and snippets are local-only.

## Roadmap

- [x] v1.0 — SSH workspace: terminal, hosts, keys, SFTP, forwarding, snippets, history, and monitoring
- [ ] v2.0 — Docker, Kubernetes, richer monitoring, and an AI assistant
- [ ] v3.0 — AWS, Azure, GCP, and session recording
- [ ] v4.0 — Plugin marketplace and team collaboration

## License

[MIT](LICENSE)
