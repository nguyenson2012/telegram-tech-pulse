package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ai-tech-pulse/digest/internal/usecase"
)

// SearchHandler handles semantic vector search requests.
type SearchHandler struct {
	searchUseCase *usecase.SearchUseCase
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(searchUseCase *usecase.SearchUseCase) *SearchHandler {
	return &SearchHandler{searchUseCase: searchUseCase}
}

// Search handles GET /api/v1/search?q=query_text&limit=5
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		RespondError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	limit := 5
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	results, err := h.searchUseCase.Search(r.Context(), query, limit)
	if err != nil {
		if errors.Is(err, usecase.ErrEmptySearchQuery) {
			RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]interface{}{
		"query":   query,
		"count":   len(results),
		"results": results,
	})
}
