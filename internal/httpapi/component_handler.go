package httpapi

import (
	"net/http"
	"strconv"

	"task140-reliability/internal/service"
)

// --- component handlers ---

func handleCreateComponent(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name      string `json:"name"`
			Category  string `json:"category"`
			Note      string `json:"note"`
			RatePPT   int64  `json:"failure_rate_ppt"`
		}
		if err := decodeBody(r, &body); err != nil {
			writeServiceError(w, err)
			return
		}
		c, err := svc.CreateComponent(r.Context(), body.Name, body.Category, body.Note, body.RatePPT)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, c)
	}
}

func handleListComponents(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cs, err := svc.ListComponents(r.Context())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, cs)
	}
}

func handleGetComponent(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := svc.GetComponent(r.Context(), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, c)
	}
}

func handleUpdateComponent(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name     *string `json:"name"`
			Category *string `json:"category"`
			Note     *string `json:"note"`
			Rate     *int64  `json:"failure_rate_ppt"`
		}
		if err := decodeBody(r, &body); err != nil {
			writeServiceError(w, err)
			return
		}
		c, err := svc.UpdateComponent(r.Context(), r.PathValue("id"), body.Name, body.Category, body.Note, body.Rate)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, c)
	}
}

func handleCreateFailureMode(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name       string `json:"name"`
			Effect     string `json:"effect"`
			Severity   int    `json:"severity"`
			Occurrence int    `json:"occurrence"`
			Detection  int    `json:"detection"`
		}
		if err := decodeBody(r, &body); err != nil {
			writeServiceError(w, err)
			return
		}
		fm, err := svc.CreateFailureMode(r.Context(), r.PathValue("id"), body.Name, body.Effect, body.Severity, body.Occurrence, body.Detection)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, fm)
	}
}

func handleListFailureModes(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ms, err := svc.ListFailureModes(r.Context(), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, ms)
	}
}

// itoa pointer helper (unused kept for clarity in case of int query params).
func ptrInt(s string) *int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}
