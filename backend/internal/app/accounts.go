package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"stoptime/internal/store"
)

// Directories: drivers and managers (operations.md, "Directories").
// Driver and manager creation is not audited (the audited actions are
// exactly the seven in data-model.md).

type CreateDriverInput struct {
	Name     string
	Email    string
	Password string
	Phone    string

	Document     *string
	VehicleName  *string
	VehiclePlate *string
	KmPerL       *string
}

// CreateDriver creates the app_user (role driver) and driver_profile in
// one transaction. The password is bcrypt-hashed (cost 10).
func (s *Services) CreateDriver(ctx context.Context, actor Actor, in CreateDriverInput) (store.Driver, error) {
	if err := s.allow(actor, OpCreateDriver); err != nil {
		return store.Driver{}, err
	}
	for _, f := range []struct{ field, value string }{
		{"name", in.Name}, {"email", in.Email},
		{"password", in.Password}, {"phone", in.Phone},
	} {
		if err := requireNonEmpty(f.field, f.value); err != nil {
			return store.Driver{}, err
		}
	}
	if !strings.Contains(in.Email, "@") {
		return store.Driver{}, &FieldError{Field: "email", Reason: "must contain @"}
	}
	if err := checkPassword(in.Password); err != nil {
		return store.Driver{}, err
	}
	if in.KmPerL != nil {
		r, err := parseDecimal("km_per_l", *in.KmPerL)
		if err != nil {
			return store.Driver{}, err
		}
		if r.Sign() <= 0 {
			return store.Driver{}, &FieldError{Field: "km_per_l", Reason: "must be positive"}
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return store.Driver{}, err
	}

	id := uuid.New()
	email := strings.ToLower(in.Email)
	err = s.Store.WithTx(ctx, func(tx *store.Store) error {
		if err := tx.InsertUser(ctx, store.User{
			ID: id, Name: in.Name, Email: email, Phone: in.Phone,
			PasswordHash: string(hash), Role: "driver",
		}); err != nil {
			return err
		}
		return tx.InsertDriverProfile(ctx, store.DriverProfile{
			UserID: id, Document: in.Document, VehicleName: in.VehicleName,
			VehiclePlate: in.VehiclePlate, KmPerL: in.KmPerL,
		})
	})
	if err != nil {
		return store.Driver{}, mapErr(err)
	}
	d, err := s.Store.DriverByID(ctx, id)
	return viewDriver(actor, d), mapErr(err)
}

// MinPasswordLength is the shortest password CreateDriver and
// CreateManager accept.
const MinPasswordLength = 8

func checkPassword(pw string) error {
	if len([]rune(pw)) < MinPasswordLength {
		return &FieldError{Field: "password", Reason: "must have at least 8 characters"}
	}
	return nil
}

type UpdateDriverInput struct {
	DriverID uuid.UUID
	Name     *string
	Phone    *string
	Active   *bool

	Document     *string
	VehicleName  *string
	VehiclePlate *string
	KmPerL       *string

	// Clear sets optional profile fields back to empty (NULL); it wins
	// over a value given for the same field. A cleared km_per_l falls
	// back to the default_km_per_l parameter (RN07).
	Clear DriverClear
}

// DriverClear names the optional profile fields to clear.
type DriverClear = store.ProfileClear

// UpdateDriver applies a partial update; nil fields are unchanged,
// Clear fields become empty. Setting active=false is the LGPD
// deactivation path.
func (s *Services) UpdateDriver(ctx context.Context, actor Actor, in UpdateDriverInput) (store.Driver, error) {
	if err := s.allow(actor, OpUpdateDriver); err != nil {
		return store.Driver{}, err
	}
	current, err := s.Store.DriverByID(ctx, in.DriverID)
	if err != nil {
		return store.Driver{}, mapErr(err)
	}
	// The edit form round-trips what the actor saw: a masked document
	// equal to the current one's mask means "unchanged"; any other value
	// with a mask character cannot be a real document.
	if in.Document != nil && strings.Contains(*in.Document, maskChar) {
		if current.Document == nil || *in.Document != maskDocument(*current.Document) {
			return store.Driver{}, &FieldError{Field: "document", Reason: "masked value; type the full document"}
		}
		in.Document = nil
	}
	if in.KmPerL != nil && !in.Clear.KmPerL {
		r, err := parseDecimal("km_per_l", *in.KmPerL)
		if err != nil {
			return store.Driver{}, err
		}
		if r.Sign() <= 0 {
			return store.Driver{}, &FieldError{Field: "km_per_l", Reason: "must be positive"}
		}
	}
	err = s.Store.WithTx(ctx, func(tx *store.Store) error {
		if err := tx.UpdateUserFields(ctx, in.DriverID, in.Name, in.Phone, in.Active); err != nil {
			return err
		}
		return tx.UpdateDriverProfileFields(ctx, in.DriverID,
			in.Document, in.VehicleName, in.VehiclePlate, in.KmPerL, in.Clear)
	})
	if err != nil {
		return store.Driver{}, mapErr(err)
	}
	d, err := s.Store.DriverByID(ctx, in.DriverID)
	return viewDriver(actor, d), mapErr(err)
}

// ListDrivers lists drivers; activeOnly hides deactivated ones.
func (s *Services) ListDrivers(ctx context.Context, actor Actor, activeOnly bool) ([]store.Driver, error) {
	if err := s.allow(actor, OpListDrivers); err != nil {
		return nil, err
	}
	drivers, err := s.Store.ListDrivers(ctx, activeOnly)
	for i := range drivers {
		drivers[i] = viewDriver(actor, drivers[i])
	}
	return drivers, mapErr(err)
}

// maskChar replaces every document digit but the last two in masked views.
const maskChar = "*"

// maskDocument keeps the separators and the last two digits (the CPF
// check digits): "123.456.789-11" → "***.***.***-11".
func maskDocument(doc string) string {
	digits := 0
	for _, r := range doc {
		if unicode.IsDigit(r) {
			digits++
		}
	}
	var b strings.Builder
	seen := 0
	for _, r := range doc {
		if unicode.IsDigit(r) {
			seen++
			if seen <= digits-2 {
				b.WriteString(maskChar)
				continue
			}
		}
		b.WriteRune(r)
	}
	return b.String()
}

// viewDriver applies the RNF06 access rule to a driver read: managers
// see the document masked; admins (and the driver, via their own
// session) see it in full.
func viewDriver(actor Actor, d store.Driver) store.Driver {
	if actor.Role == "manager" && d.Document != nil {
		masked := maskDocument(*d.Document)
		d.Document = &masked
		d.DocumentMasked = true
	}
	return d
}

// AnonymizeDriver is the RNF06 full erasure: in one transaction the
// driver's name, email, phone, password, document and vehicle
// identifiers are replaced by placeholders or cleared, the account is
// deactivated, and an `anonymize` audit row records WHICH fields were
// cleared (never their values). Routes, stops and aggregates stay. A
// driver already anonymized answers as-is, without a second audit row.
func (s *Services) AnonymizeDriver(ctx context.Context, actor Actor, driverID uuid.UUID) (store.Driver, error) {
	if err := s.allow(actor, OpAnonymizeDriver); err != nil {
		return store.Driver{}, err
	}
	current, err := s.Store.DriverByID(ctx, driverID)
	if err != nil {
		return store.Driver{}, mapErr(err)
	}
	email := "removido-" + driverID.String() + "@anonimo.invalid"
	if current.Email == email {
		return viewDriver(actor, current), nil
	}
	name := "Motorista removido " + driverID.String()[:8]
	// Not a bcrypt hash: CompareHashAndPassword always fails on it.
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return store.Driver{}, err
	}
	unusable := "!anonymized:" + hex.EncodeToString(secret)

	err = s.Store.WithTx(ctx, func(tx *store.Store) error {
		if err := tx.AnonymizeDriver(ctx, driverID, name, email, unusable); err != nil {
			return err
		}
		return tx.InsertAudit(ctx, store.AuditEntry{
			ActorUserID: actor.UserID,
			Entity:      "app_user",
			EntityID:    driverID.String(),
			Action:      "anonymize",
			OldValues: map[string]any{
				"cleared": []string{"name", "email", "phone", "password_hash",
					"document", "vehicle_name", "vehicle_plate"},
				"active": current.Active,
			},
			NewValues: map[string]any{"name": name, "email": email, "active": false},
		})
	})
	if err != nil {
		return store.Driver{}, mapErr(err)
	}
	d, err := s.Store.DriverByID(ctx, driverID)
	return viewDriver(actor, d), mapErr(err)
}

type CreateManagerInput struct {
	Name     string
	Email    string
	Password string
	Phone    string
}

func (s *Services) CreateManager(ctx context.Context, actor Actor, in CreateManagerInput) (store.User, error) {
	if err := s.allow(actor, OpCreateManager); err != nil {
		return store.User{}, err
	}
	for _, f := range []struct{ field, value string }{
		{"name", in.Name}, {"email", in.Email},
		{"password", in.Password}, {"phone", in.Phone},
	} {
		if err := requireNonEmpty(f.field, f.value); err != nil {
			return store.User{}, err
		}
	}
	if !strings.Contains(in.Email, "@") {
		return store.User{}, &FieldError{Field: "email", Reason: "must contain @"}
	}
	if err := checkPassword(in.Password); err != nil {
		return store.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return store.User{}, err
	}
	u := store.User{
		ID: uuid.New(), Name: in.Name, Email: strings.ToLower(in.Email),
		Phone: in.Phone, PasswordHash: string(hash), Role: "manager",
	}
	if err := s.Store.InsertUser(ctx, u); err != nil {
		return store.User{}, mapErr(err)
	}
	return u, nil
}

// ListManagers lists manager accounts.
func (s *Services) ListManagers(ctx context.Context, actor Actor) ([]store.User, error) {
	if err := s.allow(actor, OpListManagers); err != nil {
		return nil, err
	}
	managers, err := s.Store.ListManagers(ctx)
	return managers, mapErr(err)
}
