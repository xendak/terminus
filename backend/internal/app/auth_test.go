package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"stoptime/internal/app"
)

// Authentication (T6): login against the golden seed's demo users
// (password "stoptime-dev"), and the session cookie's HMAC lifecycle.

func TestLoginSuccess(t *testing.T) {
	u, sess, cookie, err := svc.Login(context.Background(), app.LoginInput{
		Email: "admin@stoptime.dev", Password: "stoptime-dev",
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if u.Role != "admin" || u.Email != "admin@stoptime.dev" {
		t.Errorf("user = %+v", u)
	}
	if sess.UserID != u.ID || sess.Role != "admin" {
		t.Errorf("session = %+v", sess)
	}
	// The cookie decodes (same key, any time inside the 12h window) to
	// the same session.
	got, err := app.DecodeSession(svc.SessionKey, cookie, time.Now())
	if err != nil {
		t.Fatalf("DecodeSession: %v", err)
	}
	if got.UserID != sess.UserID || got.Role != sess.Role || !got.ExpiresAt.Equal(sess.ExpiresAt) {
		t.Errorf("round-trip = %+v, want %+v", got, sess)
	}
	// 12h lifetime (architecture.md Security), measured from the
	// service clock.
	if d := time.Until(sess.ExpiresAt); d < 11*time.Hour || d > 12*time.Hour {
		t.Errorf("lifetime = %s, want ~12h", d)
	}
}

func TestLoginFailures(t *testing.T) {
	_, _, _, err := svc.Login(context.Background(), app.LoginInput{
		Email: "admin@stoptime.dev", Password: "wrong",
	})
	assertErrIs(t, "wrong password", err, app.ErrUnauthenticated)

	_, _, _, err = svc.Login(context.Background(), app.LoginInput{
		Email: "nobody@stoptime.dev", Password: "stoptime-dev",
	})
	assertErrIs(t, "unknown email", err, app.ErrUnauthenticated)

	// Deactivated accounts cannot log in (LGPD removal path).
	drv := createDriver(t, "driver-inactive")
	inactive := false
	if _, err := svc.UpdateDriver(context.Background(), adminActor(), app.UpdateDriverInput{
		DriverID: drv.ID, Active: &inactive,
	}); err != nil {
		t.Fatal(err)
	}
	_, _, _, err = svc.Login(context.Background(), app.LoginInput{
		Email: drv.Email, Password: "pw-driver-inactive",
	})
	assertErrIs(t, "inactive account", err, app.ErrUnauthenticated)
}

func TestLogout(t *testing.T) {
	if err := svc.Logout(context.Background(), adminActor()); err != nil {
		t.Errorf("Logout as admin: %v", err)
	}
	err := svc.Logout(context.Background(), app.Actor{})
	assertErrIs(t, "Logout anonymous", err, app.ErrUnauthenticated)
}

func TestSessionCookieVerification(t *testing.T) {
	key := []byte("test session key — 32+ bytes — stoptime")
		sess := app.Session{
		UserID:    admin,
		Role:      "admin",
		ExpiresAt: time.Now().Add(app.SessionLifetime),
	}
	cookie, err := app.EncodeSession(key, sess)
	if err != nil {
		t.Fatal(err)
	}

	if got, err := app.DecodeSession(key, cookie, time.Now()); err != nil || got.UserID != admin {
		t.Errorf("round-trip: %+v %v", got, err)
	}

	// Wrong key: the signature does not verify.
	if _, err := app.DecodeSession([]byte("another session key — 32 bytes ok!!"), cookie, time.Now()); !errors.Is(err, app.ErrUnauthenticated) {
		t.Errorf("wrong key = %v, want ErrUnauthenticated", err)
	}

	// Tampered payload: signature does not verify.
	if _, err := app.DecodeSession(key, "x."+cookie[len(cookie)-30:], time.Now()); !errors.Is(err, app.ErrUnauthenticated) {
		t.Errorf("tampered = %v, want ErrUnauthenticated", err)
	}

	// Expired.
	expired := sess
	expired.ExpiresAt = time.Now().Add(-time.Minute)
	stale, _ := app.EncodeSession(key, expired)
	if _, err := app.DecodeSession(key, stale, time.Now()); !errors.Is(err, app.ErrUnauthenticated) {
		t.Errorf("expired = %v, want ErrUnauthenticated", err)
	}

	// Garbage.
	for _, bad := range []string{"", "a.b", "!!!.???", "aGVsbG8"} {
		if _, err := app.DecodeSession(key, bad, time.Now()); !errors.Is(err, app.ErrUnauthenticated) {
			t.Errorf("DecodeSession(%q) = %v, want ErrUnauthenticated", bad, err)
		}
	}
}
