package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/drafthaus/drafthaus/internal/draft"
)

const sessionCookieName = "dh_session"
const sessionLifetime = 24 * time.Hour

type session struct {
	username  string
	createdAt time.Time
	expiresAt time.Time
}

// SessionStore holds all active sessions in memory.
type SessionStore struct {
	mu       sync.Mutex
	sessions map[string]*session
}

// NewSessionStore creates an empty SessionStore.
func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: make(map[string]*session)}
}

func (ss *SessionStore) create(username string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	token := hex.EncodeToString(buf)
	now := time.Now()
	ss.mu.Lock()
	ss.sessions[token] = &session{
		username:  username,
		createdAt: now,
		expiresAt: now.Add(sessionLifetime),
	}
	ss.mu.Unlock()
	return token, nil
}

func (ss *SessionStore) get(token string) (*session, bool) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	s, ok := ss.sessions[token]
	if !ok || time.Now().After(s.expiresAt) {
		delete(ss.sessions, token)
		return nil, false
	}
	return s, true
}

func (ss *SessionStore) delete(token string) {
	ss.mu.Lock()
	delete(ss.sessions, token)
	ss.mu.Unlock()
}

func sessionToken(r *http.Request) string {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionLifetime.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// AuthMiddleware protects /_admin and /_api/ paths.
// If no admin users exist yet, all access is permitted (first-run).
func AuthMiddleware(store draft.Store, sessions *SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			isAdminPath := strings.HasPrefix(path, "/_admin")
			isAPIPath := strings.HasPrefix(path, "/_api/")

			if !isAdminPath && !isAPIPath {
				next.ServeHTTP(w, r)
				return
			}

			// Allow login/setup pages without auth.
			if path == "/_admin/login" || path == "/_admin/setup" {
				next.ServeHTTP(w, r)
				return
			}

			// First-run: no admin users yet — allow all.
			hasUsers, err := store.HasAdminUsers()
			if err == nil && !hasUsers {
				next.ServeHTTP(w, r)
				return
			}

			token := sessionToken(r)
			_, ok := sessions.get(token)
			if ok {
				next.ServeHTTP(w, r)
				return
			}

			if isAPIPath {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"}) //nolint:errcheck
				return
			}

			http.Redirect(w, r, "/_admin/login", http.StatusFound)
		})
	}
}

// HandleLogin serves the login page (GET) and processes credentials (POST).
func HandleLogin(store draft.Store, sessions *SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, loginPageHTML)
		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			username := r.FormValue("username")
			password := r.FormValue("password")
			ok, err := store.ValidateCredentials(username, password)
			if err != nil || !ok {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, buildErrorPage(loginPageErrorHTML, "Invalid username or password.")) //nolint:errcheck
				return
			}
			token, err := sessions.create(username)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			setSessionCookie(w, token)
			http.Redirect(w, r, "/_admin", http.StatusFound)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandleLogout clears the session and redirects to login.
func HandleLogout(sessions *SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := sessionToken(r)
		if token != "" {
			sessions.delete(token)
		}
		clearSessionCookie(w)
		http.Redirect(w, r, "/_admin/login", http.StatusFound)
	}
}

// HandleSetup serves the first-run setup page (GET) and creates the first admin user (POST).
func HandleSetup(store draft.Store, sessions *SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If admin users already exist, redirect to login.
		hasUsers, err := store.HasAdminUsers()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if hasUsers {
			http.Redirect(w, r, "/_admin/login", http.StatusFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, setupPageHTML)
		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			username := r.FormValue("username")
			password := r.FormValue("password")
			confirm := r.FormValue("confirm")
			if username == "" || password == "" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, buildErrorPage(setupPageErrorHTML, "Username and password are required.")) //nolint:errcheck
				return
			}
			if password != confirm {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, buildErrorPage(setupPageErrorHTML, "Passwords do not match.")) //nolint:errcheck
				return
			}
			if err := store.CreateAdminUser(username, password); err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			token, err := sessions.create(username)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			setSessionCookie(w, token)
			http.Redirect(w, r, "/_admin", http.StatusFound)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

const authCSS = `*,*::before,*::after{box-sizing:border-box}
body{font-family:sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f9fafb;color:#111827}
.box{width:100%;max-width:22rem;padding:2rem;background:#fff;border-radius:0.75rem;box-shadow:0 1px 4px rgba(0,0,0,.08),0 4px 16px rgba(0,0,0,.06)}
.brand{font-size:1.25rem;font-weight:700;margin-bottom:1.75rem;text-align:center;letter-spacing:-0.02em}
.brand span{opacity:.4}
label{display:block;font-size:0.8125rem;font-weight:500;margin-bottom:0.25rem;color:#374151}
input{width:100%;padding:0.5rem 0.75rem;border:1px solid #d1d5db;border-radius:0.5rem;font:inherit;font-size:0.9375rem;margin-bottom:1rem;outline:none;transition:border-color .15s}
input:focus{border-color:#6366f1}
button{width:100%;padding:0.625rem;background:#111827;color:#fff;border:none;border-radius:0.5rem;font:inherit;font-size:0.9375rem;font-weight:600;cursor:pointer;transition:opacity .15s}
button:hover{opacity:.85}
.error{background:#fef2f2;color:#991b1b;padding:0.625rem 0.875rem;border-radius:0.5rem;font-size:0.875rem;margin-bottom:1rem}
.sub{font-size:0.8125rem;text-align:center;margin-top:1rem;color:#6b7280}`

const loginPageHTML = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>Sign In — Drafthaus</title>
<style>` + authCSS + `</style></head>
<body><div class="box">
<div class="brand">Drafthaus <span>admin</span></div>
<form method="POST" action="/_admin/login">
<label for="username">Username</label>
<input id="username" name="username" type="text" autocomplete="username" autofocus required>
<label for="password">Password</label>
<input id="password" name="password" type="password" autocomplete="current-password" required>
<button type="submit">Sign In</button>
</form>
</div></body></html>`

const loginPageErrorHTML = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>Sign In — Drafthaus</title>
<style>` + authCSS + `</style></head>
<body><div class="box">
<div class="brand">Drafthaus <span>admin</span></div>
<div class="error">{{ERROR}}</div>
<form method="POST" action="/_admin/login">
<label for="username">Username</label>
<input id="username" name="username" type="text" autocomplete="username" autofocus required>
<label for="password">Password</label>
<input id="password" name="password" type="password" autocomplete="current-password" required>
<button type="submit">Sign In</button>
</form>
</div></body></html>`

const setupPageHTML = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>Setup — Drafthaus</title>
<style>` + authCSS + `</style></head>
<body><div class="box">
<div class="brand">Drafthaus <span>setup</span></div>
<p style="font-size:.875rem;color:#6b7280;text-align:center;margin-bottom:1.25rem">Welcome to Drafthaus — create your admin account.</p>
<form method="POST" action="/_admin/setup">
<label for="username">Username</label>
<input id="username" name="username" type="text" autocomplete="username" autofocus required>
<label for="password">Password</label>
<input id="password" name="password" type="password" autocomplete="new-password" required>
<label for="confirm">Confirm Password</label>
<input id="confirm" name="confirm" type="password" autocomplete="new-password" required>
<button type="submit">Create Account</button>
</form>
</div></body></html>`

const setupPageErrorHTML = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>Setup — Drafthaus</title>
<style>` + authCSS + `</style></head>
<body><div class="box">
<div class="brand">Drafthaus <span>setup</span></div>
<p style="font-size:.875rem;color:#6b7280;text-align:center;margin-bottom:1.25rem">Welcome to Drafthaus — create your admin account.</p>
<div class="error">{{ERROR}}</div>
<form method="POST" action="/_admin/setup">
<label for="username">Username</label>
<input id="username" name="username" type="text" autocomplete="username" autofocus required>
<label for="password">Password</label>
<input id="password" name="password" type="password" autocomplete="new-password" required>
<label for="confirm">Confirm Password</label>
<input id="confirm" name="confirm" type="password" autocomplete="new-password" required>
<button type="submit">Create Account</button>
</form>
</div></body></html>`

// buildErrorPage replaces the {{ERROR}} placeholder in an error page template.
func buildErrorPage(tmpl, msg string) string {
	return strings.ReplaceAll(tmpl, "{{ERROR}}", msg)
}
