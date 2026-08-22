// Package httpapi wires the reliability service to HTTP/JSON endpoints. The
// handlers are plain functions over the service so the same mux is shared by
// the real server and the smoke test (httptest).
package httpapi

import (
	"net/http"

	"task140-reliability/internal/service"
)

// Version is the service identifier reported by /healthz and /version.
const Version = "task140-reliability/v1"

// Services bundles the service the mux wires up.
type Services struct {
	Svc *service.Service
}

// NewMux builds the HTTP handler tree over the given service. webFS serves the
// embedded frontend (nil disables the static routes).
func NewMux(svc Services, webFS http.FileSystem) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handleHealth)
	mux.HandleFunc("GET /version", handleVersion)

	// component library
	mux.HandleFunc("POST /components", handleCreateComponent(svc.Svc))
	mux.HandleFunc("GET /components", handleListComponents(svc.Svc))
	mux.HandleFunc("GET /components/{id}", handleGetComponent(svc.Svc))
	mux.HandleFunc("PATCH /components/{id}", handleUpdateComponent(svc.Svc))
	mux.HandleFunc("POST /components/{id}/failure-modes", handleCreateFailureMode(svc.Svc))
	mux.HandleFunc("GET /components/{id}/failure-modes", handleListFailureModes(svc.Svc))

	// analyses
	mux.HandleFunc("POST /analyses", handleCreateAnalysis(svc.Svc))
	mux.HandleFunc("GET /analyses", handleListAnalyses(svc.Svc))
	mux.HandleFunc("GET /analyses/{id}", handleGetAnalysis(svc.Svc))
	mux.HandleFunc("PATCH /analyses/{id}", handleUpdateAnalysisParams(svc.Svc))
	mux.HandleFunc("POST /analyses/{id}/solve", handleSolve(svc.Svc))
	mux.HandleFunc("POST /analyses/{id}/review", handleReview(svc.Svc))
	mux.HandleFunc("POST /analyses/{id}/baseline", handleBaseline(svc.Svc))
	mux.HandleFunc("POST /analyses/{id}/revise", handleRevise(svc.Svc))
	mux.HandleFunc("GET /analyses/{id}/result", handleGetResult(svc.Svc))
	mux.HandleFunc("GET /analyses/{id}/report", handleReport(svc.Svc))
	mux.HandleFunc("GET /analyses/{id}/revisions", handleListRevisions(svc.Svc))

	// fta
	mux.HandleFunc("POST /analyses/{id}/fta", handleSaveFTATree(svc.Svc))
	mux.HandleFunc("GET /analyses/{id}/fta", handleLoadFTATree(svc.Svc))

	// fmea
	mux.HandleFunc("POST /analyses/{id}/fmea/rows", handleAddFMEARow(svc.Svc))
	mux.HandleFunc("GET /analyses/{id}/fmea", handleLoadFMEATable(svc.Svc))
	mux.HandleFunc("PATCH /analyses/{id}/fmea/rows/{row}", handleUpdateFMEARow(svc.Svc))
	mux.HandleFunc("DELETE /analyses/{id}/fmea/rows/{row}", handleDeleteFMEARow(svc.Svc))

	// rbd
	mux.HandleFunc("POST /analyses/{id}/rbd", handleSaveRBDDiagram(svc.Svc))
	mux.HandleFunc("GET /analyses/{id}/rbd", handleLoadRBDDiagram(svc.Svc))

	// failure data
	mux.HandleFunc("POST /analyses/{id}/failure-events", handleCreateFailureEvent(svc.Svc))
	mux.HandleFunc("GET /analyses/{id}/failure-events", handleListFailureEvents(svc.Svc))
	mux.HandleFunc("GET /analyses/{id}/failure-events/fit", handleFitFailureData(svc.Svc))

	// reports
	mux.HandleFunc("GET /reports/analyses", handleListAnalyses(svc.Svc))

	// frontend (last, catches "/")
	if webFS != nil {
		registerFrontend(mux, webFS)
	}
	return mux
}
