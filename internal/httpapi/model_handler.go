package httpapi

import (
	"net/http"

	"task140-reliability/internal/domain"
	"task140-reliability/internal/service"
)

// --- fault tree ---

func handleSaveFTATree(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var tree domain.FTATree
		if err := decodeBody(r, &tree); err != nil {
			writeServiceError(w, err)
			return
		}
		tree.AnalysisID = r.PathValue("id")
		t, err := svc.SaveFTATree(r.Context(), tree)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, t)
	}
}

func handleLoadFTATree(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := svc.LoadFTATree(r.Context(), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, t)
	}
}

// --- fmea ---

func handleAddFMEARow(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var row domain.FMEARow
		if err := decodeBody(r, &row); err != nil {
			writeServiceError(w, err)
			return
		}
		row.AnalysisID = r.PathValue("id")
		out, err := svc.AddFMEARow(r.Context(), row)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

func handleLoadFMEATable(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tb, err := svc.LoadFMEATable(r.Context(), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, tb)
	}
}

func handleUpdateFMEARow(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Function    *string `json:"function"`
			FailureMode *string `json:"failure_mode"`
			Effect      *string `json:"effect"`
			Severity    *int    `json:"severity"`
			Occurrence  *int    `json:"occurrence"`
			Detection   *int    `json:"detection"`
			Action      *string `json:"action"`
			ActionState *string `json:"action_state"`
		}
		if err := decodeBody(r, &body); err != nil {
			writeServiceError(w, err)
			return
		}
		out, err := svc.UpdateFMEARow(r.Context(), r.PathValue("row"), body.Function, body.FailureMode, body.Effect, body.Severity, body.Occurrence, body.Detection, body.Action, body.ActionState)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleDeleteFMEARow(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := svc.DeleteFMEARow(r.Context(), r.PathValue("row")); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// --- rbd ---

func handleSaveRBDDiagram(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var diag domain.RBDDiagram
		if err := decodeBody(r, &diag); err != nil {
			writeServiceError(w, err)
			return
		}
		diag.AnalysisID = r.PathValue("id")
		out, err := svc.SaveRBDDiagram(r.Context(), diag)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleLoadRBDDiagram(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		d, err := svc.LoadRBDDiagram(r.Context(), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, d)
	}
}

// --- failure data ---

func handleCreateFailureEvent(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var e domain.FailureEvent
		if err := decodeBody(r, &e); err != nil {
			writeServiceError(w, err)
			return
		}
		e.AnalysisID = r.PathValue("id")
		out, err := svc.CreateFailureEvent(r.Context(), e)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

func handleListFailureEvents(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ev, err := svc.LoadFailureEvents(r.Context(), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, ev)
	}
}

func handleFitFailureData(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fit, err := svc.FitFailureData(r.Context(), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, fit)
	}
}
