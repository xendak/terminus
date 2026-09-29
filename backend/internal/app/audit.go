package app

import (
	"context"

	"github.com/google/uuid"

	"stoptime/internal/store"
)

// Audit (operations.md ListAudit): the append-only trail of changes to
// points and times (RNF05). Admin only per the role matrix.

type ListAuditInput struct {
	Entity *string
	From   *string
	To     *string
}

func (s *Services) ListAudit(ctx context.Context, actor Actor, in ListAuditInput) ([]store.AuditEntryView, error) {
	if err := s.allow(actor, OpListAudit); err != nil {
		return nil, err
	}
	if in.From != nil {
		f, err := parseDate("from", *in.From)
		if err != nil {
			return nil, err
		}
		in.From = &f
	}
	if in.To != nil {
		t, err := parseDate("to", *in.To)
		if err != nil {
			return nil, err
		}
		in.To = &t
	}
	entries, err := s.Store.ListAudit(ctx, in.Entity, in.From, in.To)
	return entries, mapErr(err)
}

// ExportPeriodCSV returns the CSV source rows for a window (RF12).
// Drivers export their own data only — the query-level filter, never
// post-filtered.
// ExportPeriodCSV: managerUserID optionally keeps one manager's team.
func (s *Services) ExportPeriodCSV(ctx context.Context, actor Actor, from, to string, driverUserID, managerUserID *uuid.UUID) ([]store.ExportRow, error) {
	if err := s.allow(actor, OpExportPeriodCSV); err != nil {
		return nil, err
	}
	f, err := parseDate("from", from)
	if err != nil {
		return nil, err
	}
	t, err := parseDate("to", to)
	if err != nil {
		return nil, err
	}
	if f > t {
		return nil, &FieldError{Field: "from", Reason: "window start after end"}
	}
	if actor.Role == "driver" {
		id := actor.UserID
		driverUserID = &id
	}
	rows, err := s.Store.ExportRows(ctx, f, t, driverUserID, managerUserID)
	return rows, mapErr(err)
}
