package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"vectorDemo/internal/model"
	"vectorDemo/internal/repository"
	"vectorDemo/internal/service"
)

type Handler struct {
	repo         *repository.ImageRepository
	embeddingSvc *service.EmbeddingService
}

func NewHandler(repo *repository.ImageRepository, embeddingSvc *service.EmbeddingService) *Handler {
	return &Handler{
		repo:         repo,
		embeddingSvc: embeddingSvc,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/health":
		h.Health(w, r)
	case "/api/images":
		if r.Method == http.MethodPost {
			h.UploadImage(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/api/images/search":
		if r.Method == http.MethodPost {
			h.SearchImage(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req model.UploadImageRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON format")
		return
	}

	if req.ImageURL == "" {
		writeJSONError(w, http.StatusBadRequest, "image_url is required")
		return
	}

	// Get image embedding from Jina API
	embedding, err := h.embeddingSvc.GetImageEmbedding(req.ImageURL)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to get embedding: "+err.Error())
		return
	}

	vectorStr := h.embeddingSvc.VectorToString(embedding)

	image := &model.Image{
		ImageURL:    req.ImageURL,
		Description: req.Description,
		Vector:      vectorStr,
	}

	id, err := h.repo.InsertImage(image)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to save image: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":          id,
		"image_url":   req.ImageURL,
		"description": req.Description,
	})
}

func (h *Handler) SearchImage(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req model.SearchImageRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON format")
		return
	}

	if req.ImageURL == "" {
		writeJSONError(w, http.StatusBadRequest, "image_url is required")
		return
	}

	// Get query image embedding from Jina API
	embedding, err := h.embeddingSvc.GetImageEmbedding(req.ImageURL)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to get embedding: "+err.Error())
		return
	}

	vectorStr := h.embeddingSvc.VectorToString(embedding)

	// Search for similar images
	images, err := h.repo.SearchSimilarImages(vectorStr, 5)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to search images: "+err.Error())
		return
	}

	if len(images) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "no similar images found",
			"results": []model.SearchImageResponse{},
		})
		return
	}

	results := make([]model.SearchImageResponse, len(images))
	for i, img := range images {
		results[i] = model.SearchImageResponse{
			ID:          img.ID,
			Description: img.Description,
			ImageURL:    img.ImageURL,
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
	})
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}