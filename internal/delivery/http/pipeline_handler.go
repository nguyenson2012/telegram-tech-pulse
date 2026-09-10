package http

import (
	"net/http"
	"strconv"

	"github.com/ai-tech-pulse/digest/internal/usecase"
)

// PipelineHandler handles on-demand triggers for the digest pipeline.
type PipelineHandler struct {
	pipelineUseCase *usecase.PipelineUseCase
}

// NewPipelineHandler creates a new PipelineHandler.
func NewPipelineHandler(pipelineUseCase *usecase.PipelineUseCase) *PipelineHandler {
	return &PipelineHandler{pipelineUseCase: pipelineUseCase}
}

// Run handles POST /api/v1/pipeline/run?async=true
func (h *PipelineHandler) Run(w http.ResponseWriter, r *http.Request) {
	async := true
	if asyncParam := r.URL.Query().Get("async"); asyncParam != "" {
		if parsed, err := strconv.ParseBool(asyncParam); err == nil {
			async = parsed
		}
	}

	if async {
		if err := h.pipelineUseCase.TriggerPipelineAsync(r.Context()); err != nil {
			RespondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		RespondJSON(w, http.StatusAccepted, map[string]interface{}{
			"message": "Daily pipeline job enqueued successfully into JobQueue",
			"status":  "queued",
		})
		return
	}

	digest, err := h.pipelineUseCase.ExecuteDailyPipeline(r.Context())
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Daily pipeline executed synchronously",
		"top_articles": len(digest.Articles),
		"digest":       digest,
	})
}
