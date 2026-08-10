package main

import (
	"log"
	"os"

	"github.com/devterm/core/internal/db"
	"github.com/devterm/core/internal/rpc"
	"github.com/devterm/core/internal/vault"
	"github.com/devterm/core/internal/hostmgr"
	"github.com/devterm/core/internal/keymgr"
	"github.com/devterm/core/internal/sshmgr"
	"github.com/devterm/core/internal/sftpmgr"
	"github.com/devterm/core/internal/forwardmgr"
	"github.com/devterm/core/internal/historymgr"
	"github.com/devterm/core/internal/snippetmgr"
	"github.com/devterm/core/internal/monitor"
)

func main() {
	log.SetOutput(os.Stderr) // Keep stdout clean for JSON-RPC framing

	// Initialize database
	database, err := db.Open("")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := db.Migrate(database); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Initialize vault
	v, err := vault.New()
	if err != nil {
		log.Fatalf("failed to initialize vault: %v", err)
	}

	// Initialize managers
	hostMgr := hostmgr.New(database, v)
	keyMgr := keymgr.New(database, v)
	sshMgr := sshmgr.New(database, v, hostMgr)
	sftpMgr := sftpmgr.New(sshMgr)
	fwdMgr := forwardmgr.New(database, sshMgr)
	histMgr := historymgr.New(database)
	snippetMgr := snippetmgr.New(database)
	monMgr := monitor.New(sshMgr)

	// Create RPC dispatcher and register methods
	dispatcher := rpc.NewDispatcher()

	// Register all manager methods
	hostMgr.RegisterRPC(dispatcher)
	keyMgr.RegisterRPC(dispatcher)
	sshMgr.RegisterRPC(dispatcher)
	sftpMgr.RegisterRPC(dispatcher)
	fwdMgr.RegisterRPC(dispatcher)
	histMgr.RegisterRPC(dispatcher)
	snippetMgr.RegisterRPC(dispatcher)
	monMgr.RegisterRPC(dispatcher)

	// Register built-in methods
	dispatcher.Register("ping", func(params map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"pong": true}, nil
	})

	// Run the stdio JSON-RPC server
	log.Println("devterm-core starting...")
	if err := rpc.Serve(os.Stdin, os.Stdout, dispatcher); err != nil {
		log.Fatalf("RPC server error: %v", err)
	}
}
