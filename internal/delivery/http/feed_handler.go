package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
	"github.com/ai-tech-pulse/digest/internal/usecase"
	"github.com/go-chi/chi/v5"
)

// FeedHandler handles feed management HTTP requests.
type FeedHandler struct {
	feedUseCase *usecase.FeedUseCase
}

// NewFeedHandler creates a new FeedHandler.
func NewFeedHandler(feedUseCase *usecase.FeedUseCase) *FeedHandler {
	return &FeedHandler{feedUseCase: feedUseCase}
}

// AddFeed handles POST /api/v1/feeds
func (h *FeedHandler) AddFeed(w http.ResponseWriter, r *http.Request) {
	var req usecase.AddFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	feed, err := h.feedUseCase.AddFeed(r.Context(), req)
	if err != nil {
		if errors.Is(err, entity.ErrFeedURLConflict) {
			RespondError(w, http.StatusConflict, "feed URL is already registered")
			return
		}
		if errors.Is(err, entity.ErrInvalidFeedURL) || errors.Is(err, entity.ErrEmptyFeedName) {
			RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusCreated, feed)
}

// ListFeeds handles GET /api/v1/feeds
func (h *FeedHandler) ListFeeds(w http.ResponseWriter, r *http.Request) {
	feeds, err := h.feedUseCase.ListFeeds(r.Context())
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, feeds)
}

type toggleFeedRequest struct {
	IsActive bool `json:"is_active"`
}

// ToggleFeed handles PATCH /api/v1/feeds/{id}
func (h *FeedHandler) ToggleFeed(w http.ResponseWriter, r *http.Request) {
	feedID := chi.URLParam(r, "id")
	if feedID == "" {
		RespondError(w, http.StatusBadRequest, "missing feed id")
		return
	}

	var req toggleFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.feedUseCase.ToggleFeedStatus(r.Context(), feedID, req.IsActive); err != nil {
		if errors.Is(err, entity.ErrFeedNotFound) {
			RespondError(w, http.StatusNotFound, "feed not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]interface{}{
		"id":        feedID,
		"is_active": req.IsActive,
	})
}
