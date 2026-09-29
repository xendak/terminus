package app

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"stoptime/internal/store"
)

// Authentication (operations.md "Auth"): Login verifies credentials and
// issues the session cookie value; Logout is the transport dropping the
// cookie after the session check. Nothing here touches HTTP — the
// cookie string is a value like any other.

type LoginInput struct {
	Email    string
	Password string
}

// Login verifies email + password (bcrypt) against an active user and
// returns the user, the session, and the encoded cookie value the
// transport sets. Every failure — unknown email, wrong password,
// deactivated user — is ErrUnauthenticated; the reason is not
// distinguishable from the outside.
func (s *Services) Login(ctx context.Context, in LoginInput) (store.User, Session, string, error) {
	if len(s.SessionKey) < 32 {
		return store.User{}, Session{}, "",
			fmt.Errorf("session key not configured: SESSION_KEY must be 32+ bytes")
	}
	u, err := s.Store.UserByEmail(ctx, in.Email)
	if err != nil {
		return store.User{}, Session{}, "", ErrUnauthenticated
	}
	if !u.Active {
		return store.User{}, Session{}, "", ErrUnauthenticated
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
		return store.User{}, Session{}, "", ErrUnauthenticated
	}
	sess := Session{
		UserID:    u.ID,
		Role:      u.Role,
		ExpiresAt: s.Now().Add(SessionLifetime),
	}
	cookie, err := EncodeSession(s.SessionKey, sess)
	if err != nil {
		return store.User{}, Session{}, "", fmt.Errorf("encode session: %w", err)
	}
	return u, sess, cookie, nil
}

// Logout checks the session and leaves cookie clearing to the transport
// (stateless sessions: dropping the cookie IS the logout).
func (s *Services) Logout(ctx context.Context, actor Actor) error {
	return s.allow(actor, OpLogout)
}
