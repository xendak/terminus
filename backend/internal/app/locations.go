package app

import (
	"context"

	"github.com/google/uuid"

	"stoptime/internal/store"
)

// Locations (RF03): the point registry. Location CRUD is not audited
// (the audited actions are exactly the seven in data-model.md).

type CreateLocationInput struct {
	Label     string
	Address   string
	Latitude  *string
	Longitude *string
}

func (s *Services) CreateLocation(ctx context.Context, actor Actor, in CreateLocationInput) (store.Location, error) {
	if err := s.allow(actor, OpCreateLocation); err != nil {
		return store.Location{}, err
	}
	if err := requireNonEmpty("label", in.Label); err != nil {
		return store.Location{}, err
	}
	if err := requireNonEmpty("address", in.Address); err != nil {
		return store.Location{}, err
	}
	for _, c := range []struct {
		field string
		value *string
	}{
		{"latitude", in.Latitude}, {"longitude", in.Longitude},
	} {
		if c.value != nil {
			if _, err := parseDecimal(c.field, *c.value); err != nil {
				return store.Location{}, err
			}
		}
	}
	l := store.Location{
		ID: uuid.New(), Label: in.Label, Address: in.Address,
		Latitude: in.Latitude, Longitude: in.Longitude,
		CreatedBy: actor.UserID,
	}
	if err := s.Store.InsertLocation(ctx, l); err != nil {
		return store.Location{}, mapErr(err)
	}
	return l, nil
}

type UpdateLocationInput struct {
	LocationID uuid.UUID
	Label      *string
	Address    *string
	Latitude   *string
	Longitude  *string
}

// UpdateLocation applies a partial update; nil fields are unchanged.
func (s *Services) UpdateLocation(ctx context.Context, actor Actor, in UpdateLocationInput) (store.Location, error) {
	if err := s.allow(actor, OpUpdateLocation); err != nil {
		return store.Location{}, err
	}
	for _, c := range []struct {
		field string
		value *string
	}{
		{"latitude", in.Latitude}, {"longitude", in.Longitude},
	} {
		if c.value != nil {
			if _, err := parseDecimal(c.field, *c.value); err != nil {
				return store.Location{}, err
			}
		}
	}
	if err := s.Store.UpdateLocationFields(ctx, in.LocationID,
		in.Label, in.Address, in.Latitude, in.Longitude); err != nil {
		return store.Location{}, mapErr(err)
	}
	l, err := s.Store.LocationByID(ctx, in.LocationID)
	return l, mapErr(err)
}

// ListLocations lists points, optionally filtered by free text.
func (s *Services) ListLocations(ctx context.Context, actor Actor, q *string) ([]store.Location, error) {
	if err := s.allow(actor, OpListLocations); err != nil {
		return nil, err
	}
	locations, err := s.Store.ListLocations(ctx, q)
	return locations, mapErr(err)
}
