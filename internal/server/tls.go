package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/crypto/acme/autocert"
)

// TLSConfig holds TLS settings.
type TLSConfig struct {
	Enabled bool
	Domain  string
	CertDir string // directory to cache certs, default ~/.drafthaus/certs
	Email   string // ACME account email
}

// NewTLSManager creates an autocert.Manager for the given domain.
func NewTLSManager(cfg TLSConfig) (*autocert.Manager, error) {
	if cfg.Domain == "" {
		return nil, errors.New("TLS domain must not be empty")
	}
	if cfg.CertDir == "" {
		return nil, errors.New("TLS cert dir must not be empty")
	}

	mgr := &autocert.Manager{
		Cache:      autocert.DirCache(cfg.CertDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(cfg.Domain),
	}
	if cfg.Email != "" {
		mgr.Email = cfg.Email
	}
	return mgr, nil
}

// StartTLS starts an HTTPS server on :443 and an HTTP→HTTPS redirect on :80.
func StartTLS(handler http.Handler, mgr *autocert.Manager, domain string) error {
	// HTTP redirect server on :80
	redirectSrv := &http.Server{
		Addr:    ":80",
		Handler: mgr.HTTPHandler(nil),
	}
	go func() {
		if err := redirectSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("warn: HTTP redirect server: %v", err)
		}
	}()

	tlsSrv := &http.Server{
		Addr:      ":443",
		Handler:   handler,
		TLSConfig: mgr.TLSConfig(),
	}

	log.Printf("Drafthaus running on https://%s", domain)
	if err := tlsSrv.ListenAndServeTLS("", ""); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("TLS server: %w", err)
	}
	return nil
}
