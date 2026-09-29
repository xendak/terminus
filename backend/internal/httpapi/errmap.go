package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"stoptime/internal/app"
)

// Sentinel → HTTP mapping (docs/spec/operations.md, error model).
// Adapters map; they never invent statuses.
func statusFor(err error) int {
	switch {
	case errors.Is(err, app.ErrUnauthenticated):
		return http.StatusUnauthorized
	case errors.Is(err, app.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, app.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, app.ErrBadInput):
		return http.StatusBadRequest
	case errors.Is(err, app.ErrValidation), errors.Is(err, app.ErrDepartureBeforeArrival),
		errors.Is(err, app.ErrStopTimesOutOfOrder):
		return http.StatusUnprocessableEntity
	case errors.Is(err, app.ErrDriverDateConflict),
		errors.Is(err, app.ErrRouteClosed),
		errors.Is(err, app.ErrDuplicateEmail):
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}

// writeJSONError maps a service error onto the JSON transport with
// field detail when present.
func writeJSONError(w http.ResponseWriter, err error) {
	status := statusFor(err)
	body := map[string]string{"error": err.Error()}
	if status == http.StatusInternalServerError {
		// Unmapped errors are internal: log them, never echo their text
		// (architecture.md Security).
		log.Printf("httpapi: internal error: %v", err)
		body["error"] = "internal error"
	}
	var fe *app.FieldError
	if errors.As(err, &fe) {
		body["field"] = fe.Field
		body["reason"] = fe.Reason
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeJSON encodes a successful JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// formErrors turns a service error into inline form errors for the
// screens.md "invalid" state. FieldError keeps its field; the known
// non-field sentinels land on their obvious field; anything else is a
// generic form error (no internal text leaks).
func formErrors(err error) map[string]string {
	var fe *app.FieldError
	if errors.As(err, &fe) {
		return map[string]string{fe.Field: fe.Reason}
	}
	if errors.Is(err, app.ErrDuplicateEmail) {
		return map[string]string{"email": err.Error()}
	}
	if errors.Is(err, app.ErrBadInput) {
		return map[string]string{"form": err.Error()}
	}
	if statusFor(err) < http.StatusInternalServerError {
		return map[string]string{"form": err.Error()}
	}
	return map[string]string{"form": "something went wrong"}
}

// decodeJSON reads a JSON request body into an adapter DTO; malformed
// bodies are ErrBadInput (400).
func decodeJSON(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("%w: malformed JSON body", app.ErrBadInput)
	}
	return nil
}
