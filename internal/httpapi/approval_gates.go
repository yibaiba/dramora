package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type approvalGateReviewRequest struct {
	ReviewedBy string `json:"reviewed_by"`
	ReviewNote string `json:"review_note"`
}

func (api *api) listApprovalGates(w http.ResponseWriter, r *http.Request) {
	episodeID := chi.URLParam(r, "episodeId")
	if _, err := api.projectService.GetEpisode(r.Context(), episodeID); err != nil {
		writeServiceError(w, err)
		return
	}
	gates, err := api.productionService.ListApprovalGates(r.Context(), episodeID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"approval_gates": approvalGateDTOs(gates)})
}

func (api *api) seedApprovalGates(w http.ResponseWriter, r *http.Request) {
	episode, err := api.projectService.GetEpisode(r.Context(), chi.URLParam(r, "episodeId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	gates, err := api.productionService.SeedEpisodeApprovalGates(r.Context(), episode)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, Envelope{"approval_gates": approvalGateDTOs(gates)})
}

func (api *api) approveApprovalGate(w http.ResponseWriter, r *http.Request) {
	request, ok := readApprovalGateReviewRequest(w, r)
	if !ok {
		return
	}
	gate, err := api.productionService.ApproveApprovalGate(
		r.Context(),
		chi.URLParam(r, "gateId"),
		request.ReviewedBy,
		request.ReviewNote,
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"approval_gate": approvalGateDTO(gate)})
}

func (api *api) requestApprovalChanges(w http.ResponseWriter, r *http.Request) {
	request, ok := readApprovalGateReviewRequest(w, r)
	if !ok {
		return
	}
	gate, err := api.productionService.RequestApprovalChanges(
		r.Context(),
		chi.URLParam(r, "gateId"),
		request.ReviewedBy,
		request.ReviewNote,
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"approval_gate": approvalGateDTO(gate)})
}

func readApprovalGateReviewRequest(w http.ResponseWriter, r *http.Request) (approvalGateReviewRequest, bool) {
	var request approvalGateReviewRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return approvalGateReviewRequest{}, false
	}
	return request, true
}
