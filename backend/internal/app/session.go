package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Session: the HMAC-signed identity from architecture.md (Security) —
// a cookie named st_session carrying user id, role, and expiry, 12h
// lifetime, no server-side state (revocation tradeoff recorded there;
// the upgrade path is a session table behind the same middleware).

// SessionCookieName is the cookie the transport sets (T7); the name
// lives here so encode/decode and adapters share one constant.
const SessionCookieName = "st_session"

// SessionLifetime per architecture.md: 12 hours.
const SessionLifetime = 12 * time.Hour

// Session is what the cookie carries: who, acting as which role, until
// when.
type Session struct {
	UserID    uuid.UUID `json:"uid"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"exp"`
}

// EncodeSession returns the cookie value: base64url(json) + "." +
// base64url(HMAC-SHA256 over the body). A pure function — transports and
// services share it; no HTTP types here.
func EncodeSession(key []byte, s Session) (string, error) {
	payload, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(body))
	return body + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// DecodeSession verifies signature (constant time) and expiry. Any flaw
// — bad format, bad signature, expired — is ErrUnauthenticated.
func DecodeSession(key []byte, cookie string, now time.Time) (Session, error) {
	body, sig, ok := strings.Cut(cookie, ".")
	if !ok {
		return Session{}, ErrUnauthenticated
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(body))
	want := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || !hmac.Equal(want, got) {
		return Session{}, ErrUnauthenticated
	}
	payload, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return Session{}, ErrUnauthenticated
	}
	var s Session
	if err := json.Unmarshal(payload, &s); err != nil {
		return Session{}, ErrUnauthenticated
	}
	if s.UserID == uuid.Nil || s.Role == "" || !now.Before(s.ExpiresAt) {
		return Session{}, ErrUnauthenticated
	}
	return s, nil
}

// The session also rides in the context: middleware (T7) decodes the
// cookie once and stashes it; handlers read it for rendering and pass
// the Actor to services. Services authorize on the Actor — the context
// copy is for transport concerns, the enforcement stays service-side.

type ctxKeySession struct{}

// WithSession returns a context carrying the session.
func WithSession(ctx context.Context, s Session) context.Context {
	return context.WithValue(ctx, ctxKeySession{}, s)
}

// SessionFromContext returns the session middleware stashed, if any.
func SessionFromContext(ctx context.Context) (Session, bool) {
	s, ok := ctx.Value(ctxKeySession{}).(Session)
	return s, ok
}

// ActorFromSession is the bridge: middleware turns a verified session
// into the service-level Actor.
func ActorFromSession(s Session) Actor {
	return Actor{UserID: s.UserID, Role: s.Role}
}
