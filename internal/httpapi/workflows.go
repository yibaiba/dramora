package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (api *api) startStoryAnalysis(w http.ResponseWriter, r *http.Request) {
	episode, err := api.projectService.GetEpisode(r.Context(), chi.URLParam(r, "episodeId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	result, err := api.productionService.StartStoryAnalysis(r.Context(), episode)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, Envelope{
		"workflow_run":   workflowRunDTO(result.WorkflowRun),
		"generation_job": generationJobDTO(result.GenerationJob),
	})
}

func (api *api) getWorkflowRun(w http.ResponseWriter, r *http.Request) {
	run, err := api.productionService.GetWorkflowRun(r.Context(), chi.URLParam(r, "workflowRunId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"workflow_run": workflowRunDTO(run)})
}

func (api *api) listGenerationJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := api.productionService.ListGenerationJobs(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	items := make([]generationJobResponse, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, generationJobDTO(job))
	}
	writeJSON(w, http.StatusOK, Envelope{"generation_jobs": items})
}

func (api *api) getGenerationJob(w http.ResponseWriter, r *http.Request) {
	job, err := api.productionService.GetGenerationJob(r.Context(), chi.URLParam(r, "jobId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"generation_job": generationJobDTO(job)})
}
