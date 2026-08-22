package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"task140-reliability/internal/domain"
)

// jsonError is the error response body.
type jsonError struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, jsonError{Error: msg})
}

// writeServiceError maps a domain sentinel error to an HTTP status and writes it.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrStateConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrInvalidArgument):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrDependentMissing), errors.Is(err, domain.ErrInsufficientData):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrRBDCannotReduce):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// decodeBody decodes JSON into dst. Returns an invalid-argument error on bad JSON.
func decodeBody(r *http.Request, dst any) error {
	if r.Body == nil {
		return nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return errors.Join(domain.ErrInvalidArgument, err)
	}
	return nil
}

// handleHealth / handleVersion.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": Version})
}

func notFound(w http.ResponseWriter) {
	writeError(w, http.StatusNotFound, "not found")
}
