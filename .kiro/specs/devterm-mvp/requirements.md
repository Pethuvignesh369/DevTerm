# Requirements Document

## Introduction

DevTerm is a modern, cross-platform SSH client built for DevOps engineers, SREs, Cloud Engineers, and Platform teams. It combines SSH, SFTP, and infrastructure tooling into a single native desktop application, differentiating itself from legacy SSH clients through a modern UI, open-source core, and AI-assisted workflows.

This spec covers the **MVP scope** as defined in Section 5 of the DevTerm PRD (`DevTerm_PRD.md`), which corresponds to "Version 1" of the product roadmap: core SSH connectivity, host management, terminal experience, command history, command snippets, SFTP file transfer, port forwarding, SSH key management, and a basic system dashboard, along with the foundational security and cross-platform architecture needed to support them.

**Out of scope for this spec** (to be covered in later specs per the PRD roadmap): Docker management, Kubernetes management, AWS/Azure/GCP discovery, AI Assistant, advanced monitoring/notifications, session recording, plugin system, team collaboration, cloud sync, and RBAC.

Target stack: Tauri + Vue 3 + TypeScript + Tailwind CSS + ShadCN Vue (frontend), Go (backend), SQLite (storage), `golang.org/x/crypto/ssh` and `pkg/sftp` (SSH/SFTP libraries).

## Requirements

### Requirement 1: SSH Connection & Authentication

**User Story:** As a DevOps engineer, I want to connect to remote servers using multiple authentication methods, so that I can securely access hosts regardless of their authentication requirements.

#### Acceptance Criteria

1. WHEN a user initiates a connection to a saved host THEN the system SHALL establish an SSH session using the authentication method configured for that host.
2. WHEN a user selects password authentication THEN the system SHALL prompt for and submit the password without persisting it in plaintext.
3. WHEN a user selects SSH key authentication THEN the system SHALL support both password-protected and passwordless private keys.
4. IF a private key requires a passphrase THEN the system SHALL prompt for the passphrase and SHALL NOT store it in plaintext.
5. WHEN a user has an SSH agent running THEN the system SHALL support authenticating via the SSH agent without requiring the private key to be loaded directly.
6. WHEN a user configures a host THEN the system SHALL allow associating one of multiple stored identities (credential/key sets) with that host.
7. IF authentication fails THEN the system SHALL display a clear, actionable error message and SHALL NOT crash or hang the application.
8. WHEN a connection is established or its state changes THEN the system SHALL reflect the current status (connecting, connected, disconnected, error) in the UI.

### Requirement 2: Host Management

**User Story:** As a user managing many servers, I want to organize and quickly find my hosts, so that I can connect efficiently without re-entering connection details.

#### Acceptance Criteria

1. WHEN a user adds a host THEN the system SHALL persist hostname, port, username, authentication reference, tags, group, and favorite status.
2. WHEN a user organizes hosts THEN the system SHALL support assigning each host to a group.
3. WHEN a user organizes hosts THEN the system SHALL support assigning one or more tags to each host.
4. WHEN a user marks a host as favorite THEN the system SHALL surface favorited hosts in a dedicated view or filter.
5. WHEN a user searches or filters hosts THEN the system SHALL update the visible host list in real time based on name, hostname, tag, or group.
6. WHEN a user edits or deletes a saved host THEN the system SHALL update or remove the persisted record accordingly.
7. IF a user attempts to delete a host with active sessions THEN the system SHALL warn the user before proceeding.

### Requirement 3: Terminal Experience

**User Story:** As a user working across multiple servers, I want a flexible, customizable terminal UI, so that I can manage many sessions comfortably.

#### Acceptance Criteria

1. WHEN a user opens multiple sessions THEN the system SHALL display each session in its own tab.
2. WHEN a user wants to view sessions side by side THEN the system SHALL support splitting the terminal view into multiple panes.
3. WHEN a user searches within terminal output THEN the system SHALL highlight matches and allow navigation between them.
4. WHEN a user changes the terminal theme THEN the system SHALL apply the change immediately and persist the preference.
5. WHEN a user adjusts font family or size THEN the system SHALL apply the change immediately and persist the preference.
6. WHEN a terminal session receives output THEN the system SHALL render ANSI colors and control sequences without noticeable lag under normal network conditions.
7. WHEN a user closes a tab or pane with an active session THEN the system SHALL terminate the underlying SSH session cleanly.

### Requirement 4: Command History

**User Story:** As a user who repeats tasks across sessions, I want a searchable history of commands I've run, so that I can quickly reuse them.

#### Acceptance Criteria

1. WHEN a user executes a command in a terminal session THEN the system SHALL record it in that host's command history.
2. WHEN a user searches command history THEN the system SHALL filter results by substring match in real time.
3. WHEN a user selects a history entry THEN the system SHALL insert it into the active terminal input.
4. WHEN command history grows large THEN the system SHALL page or limit displayed results to maintain UI responsiveness.

### Requirement 5: Command Snippets

**User Story:** As a user with frequently used commands, I want to save and reuse them, so that I don't have to retype them.

#### Acceptance Criteria

1. WHEN a user creates a snippet THEN the system SHALL persist a title, the command text, and optional tags.
2. WHEN a user browses snippets THEN the system SHALL support filtering by title, tag, or content.
3. WHEN a user runs a snippet against an active terminal session THEN the system SHALL insert or execute it in that session.
4. WHEN a user edits or deletes a snippet THEN the system SHALL update or remove the persisted record.

### Requirement 6: SFTP File Transfer

**User Story:** As a user managing remote files, I want built-in SFTP support, so that I don't need a separate file transfer tool.

#### Acceptance Criteria

1. WHEN a user opens the file browser for a connected host THEN the system SHALL list remote directory contents via SFTP.
2. WHEN a user uploads a file THEN the system SHALL transfer it to the selected remote path and display progress.
3. WHEN a user downloads a file THEN the system SHALL transfer it to a selected local path and display progress.
4. WHEN a user renames or deletes a remote file or directory THEN the system SHALL apply the change via SFTP and reflect it in the UI.
5. WHEN a user drags and drops files between local and remote panes THEN the system SHALL initiate the corresponding upload or download.
6. IF a transfer fails or is interrupted THEN the system SHALL report the error clearly and leave any partial file in a recoverable, clearly indicated state.

### Requirement 7: Port Forwarding

**User Story:** As a user needing to access services behind a remote host, I want to configure port forwarding, so that I can reach internal resources securely.

#### Acceptance Criteria

1. WHEN a user configures a local port forward THEN the system SHALL forward the specified local port to the specified remote host/port over the SSH connection.
2. WHEN a user configures a remote port forward THEN the system SHALL forward the specified remote port to the specified local host/port.
3. WHEN a user configures dynamic (SOCKS) forwarding THEN the system SHALL expose a local SOCKS proxy tunneled through the SSH connection.
4. WHEN a forwarding rule is active THEN the system SHALL display its status and allow the user to stop it.
5. IF the underlying SSH session disconnects THEN the system SHALL stop all associated forwarding rules and reflect that in the UI.

### Requirement 8: SSH Key Management

**User Story:** As a security-conscious user, I want to generate and manage SSH keys within the app, so that I don't need external tools.

#### Acceptance Criteria

1. WHEN a user generates a new key pair THEN the system SHALL support both RSA and ED25519 key types.
2. WHEN a user generates a key THEN the system SHALL allow setting an optional passphrase.
3. WHEN a user imports an existing key THEN the system SHALL validate its format before storing a reference to it.
4. WHEN a private key is persisted THEN the system SHALL encrypt it at rest or delegate storage to the OS keychain, and SHALL NOT store it in plaintext.
5. WHEN a user views their keys THEN the system SHALL display key metadata (type, fingerprint, comment) without exposing private key material unnecessarily.

### Requirement 9: System Dashboard

**User Story:** As a user monitoring a connected server, I want an at-a-glance resource view, so that I can spot problems quickly.

#### Acceptance Criteria

1. WHEN a user views the dashboard for a connected host THEN the system SHALL display current CPU, memory, disk, network, and uptime metrics.
2. WHEN metrics are retrieved THEN the system SHALL refresh them at a regular interval without blocking terminal interaction.
3. IF metrics cannot be retrieved (e.g., missing remote tools or permissions) THEN the system SHALL display a clear degraded state instead of failing silently or crashing.

### Requirement 10: Credential & Data Security

**User Story:** As a user storing sensitive connection data, I want strong protection of my credentials, so that my systems stay secure even if my device is compromised.

#### Acceptance Criteria

1. WHEN sensitive data (passwords, passphrases, private keys) is persisted THEN the system SHALL encrypt it using AES-256.
2. WHEN the host OS provides a native keychain/credential store THEN the system SHALL integrate with it for secret storage.
3. WHEN the application operates THEN the system SHALL NOT transmit telemetry data by default.
4. WHEN the application starts THEN the system SHALL provide full offline access to locally stored hosts, keys, snippets, and history (excluding functionality that requires an active remote connection).

### Requirement 11: Cross-Platform Desktop Application Foundation

**User Story:** As a user on any major OS, I want a consistent native experience, so that my workflow doesn't change across machines.

#### Acceptance Criteria

1. WHEN the application is built THEN the system SHALL package and run on Windows, macOS, and Linux via Tauri.
2. WHEN the UI is implemented THEN the system SHALL use Vue 3, TypeScript, Tailwind CSS, and ShadCN Vue per the defined stack.
3. WHEN backend operations (SSH, SFTP, key management, monitoring) are performed THEN the system SHALL execute them in the Go backend and communicate with the UI via Tauri IPC.
4. WHEN application data (hosts, sessions, snippets, history) is persisted THEN the system SHALL store it in a local SQLite database.
