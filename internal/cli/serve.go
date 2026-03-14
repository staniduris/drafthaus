package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/drafthaus/drafthaus/internal/server"
)

// Serve starts the HTTP server for a .draft file.
func Serve(path string, host string, port int) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", path)
	}

	store, err := draft.Open(path)
	if err != nil {
		return fmt.Errorf("open draft file: %w", err)
	}
	defer store.Close()

	srv, err := server.New(store, host, port)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	select {
	case err := <-errCh:
		return err
	case <-quit:
		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}

// ServeTLS starts an HTTPS server for a .draft file using Let's Encrypt auto-TLS.
func ServeTLS(path string, tlsCfg server.TLSConfig) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", path)
	}

	store, err := draft.Open(path)
	if err != nil {
		return fmt.Errorf("open draft file: %w", err)
	}
	defer store.Close()

	// Build the server using port 443 but we bypass srv.Start() in favour
	// of server.StartTLS which manages the autocert listener.
	srv, err := server.New(store, "0.0.0.0", 443)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	mgr, err := server.NewTLSManager(tlsCfg)
	if err != nil {
		return fmt.Errorf("create TLS manager: %w", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.StartTLS(srv.Handler(), mgr, tlsCfg.Domain)
	}()

	select {
	case err := <-errCh:
		return err
	case <-quit:
		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}
