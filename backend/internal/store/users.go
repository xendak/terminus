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
INSERT INTO driver_profile (user_id, document, vehicle_name, vehicle_plate, km_per_l, manager_user_id)
VALUES ($1, $2, $3, $4, $5::numeric, $6)`

func (s *Store) InsertDriverProfile(ctx context.Context, p DriverProfile) error {
	_, err := s.db.Exec(ctx, insertDriverProfileSQL,
		p.UserID, p.Document, p.VehicleName, p.VehiclePlate, p.KmPerL, p.ManagerUserID)
	return translate(err)
}

// ProfileClear names optional profile columns to set to NULL.
type ProfileClear struct {
	Document, VehicleName, VehiclePlate, KmPerL, ManagerUserID bool
}

// UpdateDriverProfileFields applies a partial profile update; nil
// arguments leave the column unchanged, cleared columns become NULL.
func (s *Store) UpdateDriverProfileFields(ctx context.Context, userID uuid.UUID, document, vehicleName, vehiclePlate, kmPerL *string, managerUserID *uuid.UUID, clear ProfileClear) error {
	tag, err := s.db.Exec(ctx, `
UPDATE driver_profile
   SET document      = CASE WHEN $6 THEN NULL ELSE COALESCE($2, document) END,
       vehicle_name  = CASE WHEN $7 THEN NULL ELSE COALESCE($3, vehicle_name) END,
       vehicle_plate = CASE WHEN $8 THEN NULL ELSE COALESCE($4, vehicle_plate) END,
       km_per_l      = CASE WHEN $9 THEN NULL ELSE COALESCE($5::numeric, km_per_l) END,
       manager_user_id = CASE WHEN $11 THEN NULL ELSE COALESCE($10, manager_user_id) END
 WHERE user_id = $1`, userID, document, vehicleName, vehiclePlate, kmPerL,
		clear.Document, clear.VehicleName, clear.VehiclePlate, clear.KmPerL,
		managerUserID, clear.ManagerUserID)
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
       p.document, p.vehicle_name, p.vehicle_plate, p.km_per_l::text,
       p.manager_user_id, m.name
  FROM app_user u
  JOIN driver_profile p ON p.user_id = u.id
  LEFT JOIN app_user m  ON m.id = p.manager_user_id
 WHERE u.id = $1 AND u.role = 'driver'`

// DriverByID returns a driver (user + profile) or ErrNotFound.
func (s *Store) DriverByID(ctx context.Context, id uuid.UUID) (Driver, error) {
	var d Driver
	err := s.db.QueryRow(ctx, driverByIDSQL, id).
		Scan(&d.ID, &d.Name, &d.Email, &d.Phone, &d.Active, &d.Role,
			&d.Document, &d.VehicleName, &d.VehiclePlate, &d.KmPerL,
			&d.ManagerUserID, &d.ManagerName)
	if err != nil {
		return Driver{}, scanOne(err)
	}
	return d, nil
}

const listDriversSQL = `
SELECT u.id, u.name, u.email, u.phone, u.active, u.role,
       p.document, p.vehicle_name, p.vehicle_plate, p.km_per_l::text,
       p.manager_user_id, m.name
  FROM app_user u
  JOIN driver_profile p ON p.user_id = u.id
  LEFT JOIN app_user m  ON m.id = p.manager_user_id
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
			&d.Document, &d.VehicleName, &d.VehiclePlate, &d.KmPerL,
			&d.ManagerUserID, &d.ManagerName); err != nil {
			return nil, translate(err)
		}
		drivers = append(drivers, d)
	}
	return drivers, translate(rows.Err())
}

const managerColumns = `
SELECT u.id, u.name, u.email, u.phone, u.active,
       (SELECT count(*) FROM driver_profile p
          JOIN app_user d ON d.id = p.user_id AND d.active
         WHERE p.manager_user_id = u.id)::int AS team_size
  FROM app_user u`

// ListManagers never selects the password hash; team_size counts the
// active drivers whose responsible manager each one is.
func (s *Store) ListManagers(ctx context.Context) ([]Manager, error) {
	rows, err := s.db.Query(ctx, managerColumns+` WHERE u.role = 'manager' ORDER BY u.name`)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var managers []Manager
	for rows.Next() {
		var m Manager
		if err := rows.Scan(&m.ID, &m.Name, &m.Email, &m.Phone, &m.Active, &m.TeamSize); err != nil {
			return nil, translate(err)
		}
		m.Role = "manager"
		managers = append(managers, m)
	}
	return managers, translate(rows.Err())
}

// ManagerByID returns a manager account with its team size; other roles
// and unknown ids are ErrNotFound.
func (s *Store) ManagerByID(ctx context.Context, id uuid.UUID) (Manager, error) {
	var m Manager
	err := s.db.QueryRow(ctx, managerColumns+` WHERE u.id = $1 AND u.role = 'manager'`, id).
		Scan(&m.ID, &m.Name, &m.Email, &m.Phone, &m.Active, &m.TeamSize)
	if err != nil {
		return Manager{}, scanOne(err)
	}
	m.Role = "manager"
	return m, nil
}

// AnonymizeUser pseudonymizes an account of the given role (RNF06 full
// erasure): the app_user identity fields get the given placeholders and
// the account is deactivated; for drivers the profile's document and
// vehicle identifiers are cleared too. km_per_l (operational) and every
// route row stay untouched. Unknown id or other role is ErrNotFound.
func (s *Store) AnonymizeUser(ctx context.Context, id uuid.UUID, role, name, email, passwordHash string) error {
	tag, err := s.db.Exec(ctx, `
UPDATE app_user
   SET name = $2, email = $3, phone = '', password_hash = $4, active = false
 WHERE id = $1 AND role = $5`, id, name, email, passwordHash, role)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if role != "driver" {
		return nil
	}
	_, err = s.db.Exec(ctx, `
UPDATE driver_profile
   SET document = NULL, vehicle_name = NULL, vehicle_plate = NULL
 WHERE user_id = $1`, id)
	return translate(err)
}
