package store

import (
	"context"

	"github.com/google/uuid"
)

const insertUserSQL = `
INSERT INTO app_user (id, name, email, phone, password_hash, role)
VALUES ($1, $2, $3, $4, $5, $6)`

func (s *Store) InsertUser(ctx context.Context, u User) error {
	_, err := s.db.Exec(ctx, insertUserSQL,
		u.ID, u.Name, u.Email, u.Phone, u.PasswordHash, u.Role)
	return translate(err)
}

const userByEmailSQL = `
SELECT id, name, email, phone, password_hash, role, active
  FROM app_user WHERE lower(email) = lower($1)`

// UserByEmail finds an account by email case-insensitively (login).
func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.db.QueryRow(ctx, userByEmailSQL, email).
		Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash, &u.Role, &u.Active)
	if err != nil {
		return User{}, scanOne(err)
	}
	return u, nil
}

const userByIDSQL = `
SELECT id, name, email, phone, password_hash, role, active
  FROM app_user WHERE id = $1`

func (s *Store) UserByID(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	err := s.db.QueryRow(ctx, userByIDSQL, id).
		Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash, &u.Role, &u.Active)
	if err != nil {
		return User{}, scanOne(err)
	}
	return u, nil
}

// UpdateUserFields applies a partial update; nil arguments leave the
// column unchanged.
func (s *Store) UpdateUserFields(ctx context.Context, id uuid.UUID, name, phone *string, active *bool) error {
	tag, err := s.db.Exec(ctx, `
UPDATE app_user
   SET name = COALESCE($2, name),
       phone = COALESCE($3, phone),
       active = COALESCE($4, active)
 WHERE id = $1`, id, name, phone, active)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

const insertDriverProfileSQL = `
INSERT INTO driver_profile (user_id, document, vehicle_name, vehicle_plate, km_per_l)
VALUES ($1, $2, $3, $4, $5::numeric)`

func (s *Store) InsertDriverProfile(ctx context.Context, p DriverProfile) error {
	_, err := s.db.Exec(ctx, insertDriverProfileSQL,
		p.UserID, p.Document, p.VehicleName, p.VehiclePlate, p.KmPerL)
	return translate(err)
}

// UpdateDriverProfileFields applies a partial profile update; nil
// arguments leave the column unchanged.
func (s *Store) UpdateDriverProfileFields(ctx context.Context, userID uuid.UUID, document, vehicleName, vehiclePlate, kmPerL *string) error {
	tag, err := s.db.Exec(ctx, `
UPDATE driver_profile
   SET document = COALESCE($2, document),
       vehicle_name = COALESCE($3, vehicle_name),
       vehicle_plate = COALESCE($4, vehicle_plate),
       km_per_l = COALESCE($5::numeric, km_per_l)
 WHERE user_id = $1`, userID, document, vehicleName, vehiclePlate, kmPerL)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

const driverByIDSQL = `
SELECT u.id, u.name, u.email, u.phone, u.active, u.role,
       p.document, p.vehicle_name, p.vehicle_plate, p.km_per_l::text
  FROM app_user u
  JOIN driver_profile p ON p.user_id = u.id
 WHERE u.id = $1 AND u.role = 'driver'`

// DriverByID returns a driver (user + profile) or ErrNotFound.
func (s *Store) DriverByID(ctx context.Context, id uuid.UUID) (Driver, error) {
	var d Driver
	err := s.db.QueryRow(ctx, driverByIDSQL, id).
		Scan(&d.ID, &d.Name, &d.Email, &d.Phone, &d.Active, &d.Role,
			&d.Document, &d.VehicleName, &d.VehiclePlate, &d.KmPerL)
	if err != nil {
		return Driver{}, scanOne(err)
	}
	return d, nil
}

const listDriversSQL = `
SELECT u.id, u.name, u.email, u.phone, u.active, u.role,
       p.document, p.vehicle_name, p.vehicle_plate, p.km_per_l::text
  FROM app_user u
  JOIN driver_profile p ON p.user_id = u.id
 WHERE u.role = 'driver' AND (NOT $1::boolean OR u.active)
 ORDER BY u.name`

func (s *Store) ListDrivers(ctx context.Context, activeOnly bool) ([]Driver, error) {
	rows, err := s.db.Query(ctx, listDriversSQL, activeOnly)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var drivers []Driver
	for rows.Next() {
		var d Driver
		if err := rows.Scan(&d.ID, &d.Name, &d.Email, &d.Phone, &d.Active, &d.Role,
			&d.Document, &d.VehicleName, &d.VehiclePlate, &d.KmPerL); err != nil {
			return nil, translate(err)
		}
		drivers = append(drivers, d)
	}
	return drivers, translate(rows.Err())
}

const listManagersSQL = `
SELECT id, name, email, phone, active
  FROM app_user WHERE role = 'manager' ORDER BY name`

// ListManagers never selects the password hash.
func (s *Store) ListManagers(ctx context.Context) ([]User, error) {
	rows, err := s.db.Query(ctx, listManagersSQL)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var managers []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.Active); err != nil {
			return nil, translate(err)
		}
		u.Role = "manager"
		managers = append(managers, u)
	}
	return managers, translate(rows.Err())
}
