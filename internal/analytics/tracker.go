package analytics

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Tracker records page views in SQLite.
type Tracker struct {
	db *sql.DB
}

// NewTracker creates a Tracker backed by the given database.
func NewTracker(db *sql.DB) *Tracker {
	return &Tracker{db: db}
}

// SiteStats holds aggregate analytics for a time window.
type SiteStats struct {
	TotalViews     int            `json:"total_views"`
	UniqueVisitors int            `json:"unique_visitors"`
	TopPages       []PageStat     `json:"top_pages"`
	ViewsByDay     []DayStat      `json:"views_by_day"`
	TopReferrers   []ReferrerStat `json:"top_referrers"`
}

// PageStat holds view counts for a single path.
type PageStat struct {
	Path  string `json:"path"`
	Views int    `json:"views"`
}

// DayStat holds view counts for a single day.
type DayStat struct {
	Date  string `json:"date"`
	Views int    `json:"views"`
}

// ReferrerStat holds view counts from a single referrer domain.
type ReferrerStat struct {
	Referrer string `json:"referrer"`
	Views    int    `json:"views"`
}

var botSignals = []string{
	"bot", "crawler", "spider", "slurp", "bingbot", "googlebot",
	"yandexbot", "duckduckbot", "baiduspider", "facebookexternalhit",
	"twitterbot", "linkedinbot", "whatsapp", "telegram", "applebot",
	"semrushbot", "ahrefsbot", "mj12bot", "dotbot",
}

var skipPrefixes = []string{"/_admin", "/_api", "/_assets", "/_dh"}

func isBot(ua string) bool {
	lower := strings.ToLower(ua)
	for _, sig := range botSignals {
		if strings.Contains(lower, sig) {
			return true
		}
	}
	return false
}

func shouldSkip(path string) bool {
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func simplifyUA(ua string) string {
	lower := strings.ToLower(ua)
	switch {
	case strings.Contains(lower, "firefox"):
		return "Firefox"
	case strings.Contains(lower, "edg"):
		return "Edge"
	case strings.Contains(lower, "chrome"):
		return "Chrome"
	case strings.Contains(lower, "safari"):
		return "Safari"
	case strings.Contains(lower, "opera") || strings.Contains(lower, "opr"):
		return "Opera"
	case strings.Contains(lower, "curl"):
		return "curl"
	default:
		return "Other"
	}
}

func referrerDomain(ref string) string {
	if ref == "" {
		return ""
	}
	// Strip scheme.
	s := ref
	if i := strings.Index(s, "://"); i != -1 {
		s = s[i+3:]
	}
	// Strip path.
	if i := strings.Index(s, "/"); i != -1 {
		s = s[:i]
	}
	// Strip port.
	if i := strings.LastIndex(s, ":"); i != -1 {
		s = s[:i]
	}
	return s
}

func realIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// Strip port from RemoteAddr.
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i != -1 {
		return addr[:i]
	}
	return addr
}

// Track records a single page view. It is safe to call concurrently.
// Silently drops on bot detection or skipped paths — errors are non-fatal.
func (t *Tracker) Track(r *http.Request, path string) {
	ua := r.UserAgent()
	if isBot(ua) || shouldSkip(path) {
		return
	}

	ip := realIP(r)
	date := time.Now().UTC().Format("2006-01-02")
	raw := fmt.Sprintf("%s|%s|%s", ip, ua, date)
	sum := sha256.Sum256([]byte(raw))
	visitorID := hex.EncodeToString(sum[:])[:16]

	ref := referrerDomain(r.Referer())
	simpleUA := simplifyUA(ua)

	_, _ = t.db.Exec(
		`INSERT INTO page_views (path, referrer, user_agent, visitor_id) VALUES (?, ?, ?, ?)`,
		path, ref, simpleUA, visitorID,
	)
}

// Stats returns aggregate analytics for the last `days` days.
func (t *Tracker) Stats(days int) (*SiteStats, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Unix()

	stats := &SiteStats{}

	// TotalViews
	if err := t.db.QueryRow(
		`SELECT COUNT(*) FROM page_views WHERE created_at >= ?`, cutoff,
	).Scan(&stats.TotalViews); err != nil {
		return nil, fmt.Errorf("total views: %w", err)
	}

	// UniqueVisitors
	if err := t.db.QueryRow(
		`SELECT COUNT(DISTINCT visitor_id) FROM page_views WHERE created_at >= ?`, cutoff,
	).Scan(&stats.UniqueVisitors); err != nil {
		return nil, fmt.Errorf("unique visitors: %w", err)
	}

	// TopPages
	rows, err := t.db.Query(
		`SELECT path, COUNT(*) AS cnt FROM page_views WHERE created_at >= ?
		 GROUP BY path ORDER BY cnt DESC LIMIT 10`, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("top pages: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p PageStat
		if err := rows.Scan(&p.Path, &p.Views); err != nil {
			return nil, fmt.Errorf("scan top pages: %w", err)
		}
		stats.TopPages = append(stats.TopPages, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("top pages rows: %w", err)
	}

	// ViewsByDay
	drows, err := t.db.Query(
		`SELECT date(created_at, 'unixepoch') AS d, COUNT(*) AS cnt
		 FROM page_views WHERE created_at >= ?
		 GROUP BY d ORDER BY d`, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("views by day: %w", err)
	}
	defer drows.Close()
	for drows.Next() {
		var d DayStat
		if err := drows.Scan(&d.Date, &d.Views); err != nil {
			return nil, fmt.Errorf("scan views by day: %w", err)
		}
		stats.ViewsByDay = append(stats.ViewsByDay, d)
	}
	if err := drows.Err(); err != nil {
		return nil, fmt.Errorf("views by day rows: %w", err)
	}

	// TopReferrers
	rrows, err := t.db.Query(
		`SELECT referrer, COUNT(*) AS cnt FROM page_views
		 WHERE created_at >= ? AND referrer != ''
		 GROUP BY referrer ORDER BY cnt DESC LIMIT 10`, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("top referrers: %w", err)
	}
	defer rrows.Close()
	for rrows.Next() {
		var ref ReferrerStat
		if err := rrows.Scan(&ref.Referrer, &ref.Views); err != nil {
			return nil, fmt.Errorf("scan top referrers: %w", err)
		}
		stats.TopReferrers = append(stats.TopReferrers, ref)
	}
	if err := rrows.Err(); err != nil {
		return nil, fmt.Errorf("top referrers rows: %w", err)
	}

	return stats, nil
}
