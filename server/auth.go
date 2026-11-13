package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const sessionCookie = "rg_session"
const sessionTTL = 30 * 24 * time.Hour

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
}

type ctxKey struct{}

func userFrom(r *http.Request) User { return r.Context().Value(ctxKey{}).(User) }

func randomToken() string {
	b := make([]byte, 32) // 256 bits
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func hashToken(t string) []byte {
	h := sha256.Sum256([]byte(t))
	return h[:]
}

func (a *App) startSession(w http.ResponseWriter, r *http.Request, userID int64) error {
	tok := randomToken()
	exp := time.Now().Add(sessionTTL)
	if _, err := a.db.Exec("insert into sessions (token_hash, user_id, expires_at) values (?, ?, ?)",
		hashToken(tok), userID, exp.Unix()); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: tok, Path: "/", Expires: exp,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil})
	return nil
}

func (a *App) currentUser(r *http.Request) (User, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return User{}, false
	}
	var u User
	err = a.db.QueryRow(`select u.id, u.username, u.is_admin from sessions s join users u on u.id = s.user_id
		where s.token_hash = ? and s.expires_at > unixepoch()`, hashToken(c.Value)).Scan(&u.ID, &u.Username, &u.IsAdmin)
	return u, err == nil
}

// auth wraps a handler that needs a signed-in user (admin = also needs is_admin).
func (a *App) auth(admin bool, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := a.currentUser(r)
		if !ok {
			httpError(w, http.StatusUnauthorized, "Sign in first")
			return
		}
		if admin && !u.IsAdmin {
			httpError(w, http.StatusForbidden, "Admins only")
			return
		}
		h(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
	}
}

func hashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(h), err
}

// dummyHash keeps login timing the same whether or not the username exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)

func checkPassword(hash, pw string) bool {
	if hash == "" {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(pw))
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// clientIP is the visitor's address. Behind a proxy on this machine (Caddy, Tailscale Serve or Funnel) every request
// comes from loopback, so the address the proxy saw, the last X-Forwarded-For entry, is used instead; a connection
// from anywhere else can't set it.
func clientIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	if p := net.ParseIP(ip); p != nil && p.IsLoopback() {
		if parts := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ","); len(parts) > 0 {
			if last := strings.TrimSpace(parts[len(parts)-1]); last != "" {
				return last
			}
		}
	}
	return ip
}

// limiter allows n attempts per IP per minute (login, share-link unlock): stops LAN brute force.
type limiter struct {
	mu   sync.Mutex
	n    int
	hits map[string][]time.Time
}

func newLimiter(n int) *limiter { return &limiter{n: n, hits: map[string][]time.Time{}} }

func (l *limiter) allow(r *http.Request) bool {
	ip := clientIP(r)
	l.mu.Lock()
	defer l.mu.Unlock()
	cut := time.Now().Add(-time.Minute)
	recent := l.hits[ip][:0]
	for _, t := range l.hits[ip] {
		if t.After(cut) {
			recent = append(recent, t)
		}
	}
	if len(recent) >= l.n {
		l.hits[ip] = recent
		return false
	}
	l.hits[ip] = append(recent, time.Now())
	if len(l.hits) > 10000 { // bound memory
		l.hits = map[string][]time.Time{}
	}
	return true
}

func validUsername(s string) bool {
	return len(s) >= 1 && len(s) <= 64 && !strings.ContainsAny(s, " \t\n\r")
}
