package httpapi

import (
	"net/http"

	"task140-reliability/internal/domain"
	"task140-reliability/internal/service"
)

// --- analysis handlers ---

func handleCreateAnalysis(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name      string `json:"name"`
			TopEvent string `json:"top_event"`
			Params   *domain.Analysis `json:"params,omitempty"`
		}
		if err := decodeBody(r, &body); err != nil {
			writeServiceError(w, err)
			return
		}
		a, err := svc.CreateAnalysis(r.Context(), body.Name, body.TopEvent, body.Params)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, a)
	}
}

func handleListAnalyses(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		as, err := svc.ListAnalyses(r.Context())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, as)
	}
}

func handleGetAnalysis(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, err := svc.GetAnalysis(r.Context(), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, a)
	}
}

func handleUpdateAnalysisParams(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body domain.Analysis
		if err := decodeBody(r, &body); err != nil {
			writeServiceError(w, err)
			return
		}
		a, err := svc.UpdateAnalysisParams(r.Context(), r.PathValue("id"), body)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, a)
	}
}

func handleSolve(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := svc.Solve(r.Context(), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, res)
	}
}

func handleReview(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, err := svc.Review(r.Context(), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, a)
	}
}

func handleBaseline(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, err := svc.Baseline(r.Context(), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, a)
	}
}

func handleRevise(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, err := svc.Revise(r.Context(), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, a)
	}
}

func handleGetResult(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := svc.GetResult(r.Context(), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, res)
	}
}

func handleReport(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rep, err := svc.Report(r.Context(), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

func handleListRevisions(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		revs, err := svc.ListRevisions(r.Context(), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, revs)
	}
}
