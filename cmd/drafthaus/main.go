package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/drafthaus/drafthaus/internal/ai"
	"github.com/drafthaus/drafthaus/internal/cli"
	"github.com/drafthaus/drafthaus/internal/server"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "init":
		err = handleInit()
	case "serve":
		err = handleServe()
	case "export":
		err = handleExport()
	case "passwd":
		err = handlePasswd()
	case "generate":
		err = handleGenerate()
	case "serve-multi":
		err = handleServeMulti()
	case "sites":
		err = handleSites()
	case "backup":
		err = handleBackup()
	case "restore":
		err = handleRestore()
	case "import":
		err = handleImport()
	case "version":
		fmt.Println("drafthaus", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// handleInit parses: drafthaus init <name> [--template <template>]
func handleInit() error {
	args := os.Args[2:]
	if len(args) < 1 {
		return fmt.Errorf("usage: drafthaus init <name> [--template blog|cafe|portfolio|business|blank]")
	}

	name := args[0]
	template := "blog"

	for i := 1; i < len(args)-1; i++ {
		if args[i] == "--template" {
			template = args[i+1]
			i++
		}
	}

	return cli.Init(name, template)
}

// handleServe parses: drafthaus serve <file> [--port <port>] [--host <host>] [--tls --domain <d> [--email <e>] [--cert-dir <d>]]
func handleServe() error {
	args := os.Args[2:]
	if len(args) < 1 {
		return fmt.Errorf("usage: drafthaus serve <file.draft> [--port 3000] [--host 0.0.0.0] [--tls --domain example.com]")
	}

	filePath := args[0]
	host := "0.0.0.0"
	port := 3000
	tlsCfg := server.TLSConfig{}

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--port":
			if i+1 >= len(args) {
				return fmt.Errorf("--port requires a value")
			}
			p, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("invalid port %q: %w", args[i+1], err)
			}
			port = p
			i++
		case "--host":
			if i+1 >= len(args) {
				return fmt.Errorf("--host requires a value")
			}
			host = args[i+1]
			i++
		case "--tls":
			tlsCfg.Enabled = true
		case "--domain":
			if i+1 >= len(args) {
				return fmt.Errorf("--domain requires a value")
			}
			tlsCfg.Domain = args[i+1]
			i++
		case "--email":
			if i+1 >= len(args) {
				return fmt.Errorf("--email requires a value")
			}
			tlsCfg.Email = args[i+1]
			i++
		case "--cert-dir":
			if i+1 >= len(args) {
				return fmt.Errorf("--cert-dir requires a value")
			}
			tlsCfg.CertDir = args[i+1]
			i++
		}
	}

	if tlsCfg.Enabled {
		if tlsCfg.Domain == "" {
			return fmt.Errorf("--tls requires --domain <domain>")
		}
		if tlsCfg.CertDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("get home dir: %w", err)
			}
			tlsCfg.CertDir = home + "/.drafthaus/certs"
		}
		return cli.ServeTLS(filePath, tlsCfg)
	}

	return cli.Serve(filePath, host, port)
}

// handleExport parses: drafthaus export <file> <output-dir>
func handleExport() error {
	args := os.Args[2:]
	if len(args) < 2 {
		return fmt.Errorf("usage: drafthaus export <file.draft> <output-dir>")
	}
	return cli.Export(args[0], args[1])
}

// handlePasswd parses: drafthaus passwd <file> <username> <new-password>
func handlePasswd() error {
	args := os.Args[2:]
	if len(args) < 3 {
		return fmt.Errorf("usage: drafthaus passwd <file.draft> <username> <new-password>")
	}
	return cli.Passwd(args[0], args[1], args[2])
}

// handleGenerate parses: drafthaus generate <name> "<description>" [--spec file.json] [--provider openai|anthropic|ollama] [--api-key KEY]
func handleGenerate() error {
	args := os.Args[2:]
	if len(args) < 1 {
		return fmt.Errorf("usage: drafthaus generate <name> --spec <file.json>\n       drafthaus generate <name> \"<description>\" --provider anthropic --api-key KEY")
	}

	name := args[0]
	description := ""
	specFile := ""
	cfg := ai.Config{Provider: "openai"}

	if len(args) > 1 && !strings.HasPrefix(args[1], "--") {
		description = args[1]
	}

	for i := 1; i < len(args); i++ {
		if i+1 >= len(args) {
			break
		}
		switch args[i] {
		case "--spec":
			specFile = args[i+1]
			i++
		case "--provider":
			cfg.Provider = args[i+1]
			i++
		case "--api-key":
			cfg.APIKey = args[i+1]
			i++
		case "--model":
			cfg.Model = args[i+1]
			i++
		case "--base-url":
			cfg.BaseURL = args[i+1]
			i++
		}
	}

	if cfg.APIKey == "" {
		switch cfg.Provider {
		case "openai":
			cfg.APIKey = os.Getenv("OPENAI_API_KEY")
		case "anthropic":
			cfg.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		}
	}

	return cli.Generate(name, description, cfg, specFile)
}

// handleServeMulti parses: drafthaus serve-multi <directory> [--port 3000] [--host 0.0.0.0]
func handleServeMulti() error {
	args := os.Args[2:]
	if len(args) < 1 {
		return fmt.Errorf("usage: drafthaus serve-multi <directory> [--port 3000] [--host 0.0.0.0]")
	}

	dir := args[0]
	host := "0.0.0.0"
	port := 3000

	for i := 1; i < len(args)-1; i++ {
		switch args[i] {
		case "--port":
			p, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("invalid port %q: %w", args[i+1], err)
			}
			port = p
			i++
		case "--host":
			host = args[i+1]
			i++
		}
	}

	return cli.ServeMulti(dir, host, port)
}

// handleBackup parses: drafthaus backup <file.draft> [output-path]
func handleBackup() error {
	args := os.Args[2:]
	if len(args) < 1 {
		return fmt.Errorf("usage: drafthaus backup <file.draft> [output-path]")
	}
	output := ""
	if len(args) >= 2 {
		output = args[1]
	}
	return cli.Backup(args[0], output)
}

// handleRestore parses: drafthaus restore <backup.draft> <target.draft>
func handleRestore() error {
	args := os.Args[2:]
	if len(args) < 2 {
		return fmt.Errorf("usage: drafthaus restore <backup.draft> <target.draft>")
	}
	return cli.Restore(args[0], args[1])
}

// handleImport parses: drafthaus import wordpress <export.xml> <output-name>
func handleImport() error {
	args := os.Args[2:]
	if len(args) < 1 {
		return fmt.Errorf("usage: drafthaus import wordpress <export.xml> <output-name>")
	}
	switch args[0] {
	case "wordpress":
		if len(args) < 3 {
			return fmt.Errorf("usage: drafthaus import wordpress <export.xml> <output-name>")
		}
		return cli.ImportWordPress(args[1], args[2])
	default:
		return fmt.Errorf("unknown import format %q (supported: wordpress)", args[0])
	}
}

// handleSites parses: drafthaus sites <directory>
func handleSites() error {
	args := os.Args[2:]
	if len(args) < 1 {
		return fmt.Errorf("usage: drafthaus sites <directory>")
	}
	return cli.ListSites(args[0])
}

func printUsage() {
	fmt.Print(`Drafthaus v` + version + ` -- Your digital presence in one file.

Usage:
  drafthaus init <name> [--template blog|cafe|portfolio|business|blank]
  drafthaus serve <file.draft> [--port 3000] [--host 0.0.0.0]
  drafthaus serve <file.draft> --tls --domain example.com [--email you@example.com] [--cert-dir ~/.drafthaus/certs]
  drafthaus serve-multi <directory> [--port 3000] [--host 0.0.0.0]
  drafthaus sites <directory>
  drafthaus export <file.draft> <output-dir>
  drafthaus passwd <file.draft> <username> <new-password>
  drafthaus backup <file.draft> [output-path]
  drafthaus restore <backup.draft> <target.draft>
  drafthaus import wordpress <export.xml> <output-name>
  drafthaus generate <name> "<description>" [--provider openai] [--api-key KEY]
  drafthaus version

Examples:
  drafthaus init mysite --template blog
  drafthaus serve mysite.draft
  drafthaus serve mysite.draft --tls --domain mysite.com --email admin@mysite.com
  drafthaus serve-multi ./sites/
  drafthaus sites ./sites/
  drafthaus export mysite.draft ./public
  drafthaus passwd mysite.draft admin mysecurepass
  drafthaus backup mysite.draft
  drafthaus restore mysite-backup-20240115-120000.draft mysite.draft
  drafthaus import wordpress export.xml mysite
  drafthaus generate myshop "an online store for handmade ceramics" --provider openai
`)
}
