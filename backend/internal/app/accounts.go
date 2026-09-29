package app

import (
	"context"
	"strings"

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
	return d, mapErr(err)
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
	if _, err := s.Store.DriverByID(ctx, in.DriverID); err != nil {
		return store.Driver{}, mapErr(err)
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
	err := s.Store.WithTx(ctx, func(tx *store.Store) error {
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
	return d, mapErr(err)
}

// ListDrivers lists drivers; activeOnly hides deactivated ones.
func (s *Services) ListDrivers(ctx context.Context, actor Actor, activeOnly bool) ([]store.Driver, error) {
	if err := s.allow(actor, OpListDrivers); err != nil {
		return nil, err
	}
	drivers, err := s.Store.ListDrivers(ctx, activeOnly)
	return drivers, mapErr(err)
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
