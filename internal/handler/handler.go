package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vectorDemo/internal/model"
	"vectorDemo/internal/repository"
	"vectorDemo/internal/service"
)

const (
	UploadDir = "./uploads"
)

type Handler struct {
	repo         *repository.ImageRepository
	embeddingSvc service.EmbeddingServiceInterface
}

func NewHandler(repo *repository.ImageRepository, embeddingSvc service.EmbeddingServiceInterface) *Handler {
	// 创建上传目录
	if err := os.MkdirAll(UploadDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create upload directory: %v", err))
	}
	return &Handler{
		repo:         repo,
		embeddingSvc: embeddingSvc,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	
	// 处理静态文件服务（/uploads/ 目录）
	if strings.HasPrefix(path, "/uploads/") {
		fileName := strings.TrimPrefix(path, "/uploads/")
		filePath := filepath.Join(UploadDir, fileName)
		http.ServeFile(w, r, filePath)
		return
	}
	
	switch {
	case path == "/api/images" && r.Method == http.MethodPost:
		h.UploadImage(w, r)
	case path == "/api/images/search" && r.Method == http.MethodPost:
		h.SearchImage(w, r)
	case path == "/health" && r.Method == http.MethodGet:
		h.HealthCheck(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type UploadRequest struct {
	ImageURL    string `json:"image_url"`
	Description string `json:"description"`
}

type UploadResponse struct {
	ID          int64  `json:"id"`
	ImageURL    string `json:"image_url"`
	Description string `json:"description"`
}

type SearchRequest struct {
	Query string `json:"query"`
}

type SearchResult struct {
	ID         int64   `json:"id"`
	ImageURL   string  `json:"image_url"`
	Description string  `json:"description"`
	Similarity float64 `json:"similarity"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	var imageURL, description string
	var embedding []float64
	var err error

	if strings.HasPrefix(contentType, "application/json") {
		// JSON 格式上传（URL 方式）
		var req UploadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		imageURL = req.ImageURL
		description = req.Description

		// 使用文字描述生成向量
		if description == "" {
			description = "图片"
		}
		embedding, err = h.embeddingSvc.GetTextEmbedding(description)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to get embedding: "+err.Error())
			return
		}

	} else if strings.HasPrefix(contentType, "multipart/form-data") {
		// FormData 格式上传（本地文件）
		file, header, err := r.FormFile("image")
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "failed to get file: "+err.Error())
			return
		}
		defer file.Close()

		fileData, err := io.ReadAll(file)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to read file: "+err.Error())
			return
		}

		// 生成唯一文件名
		timestamp := time.Now().UnixNano()
		filename := fmt.Sprintf("%d%s", timestamp, filepath.Ext(header.Filename))
		filepath := filepath.Join(UploadDir, filename)

		// 保存文件到本地
		if err := os.WriteFile(filepath, fileData, 0644); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to save file: "+err.Error())
			return
		}

		// 构建可访问的 URL
		imageURL = fmt.Sprintf("http://localhost:8080/uploads/%s", filename)
		description = r.FormValue("description")

		// 使用本地图片文件生成向量（支持多模态嵌入）
		localFilePath := filepath
		embedding, err = h.embeddingSvc.GetImageEmbeddingFromFilePath(localFilePath)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to get image embedding: "+err.Error())
			return
		}
	}

	vectorStr := h.embeddingSvc.VectorToString(embedding)

	image := &model.Image{
		ImageURL:    imageURL,
		Description: description,
		Vector:      vectorStr,
	}

	id, err := h.repo.InsertImage(image)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to save image: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, UploadResponse{
		ID:          id,
		ImageURL:    imageURL,
		Description: description,
	})
}

// SearchImage 通过文字描述搜索相似图片
func (h *Handler) SearchImage(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Query == "" {
		writeJSONError(w, http.StatusBadRequest, "query is required")
		return
	}

	// Get text embedding
	embedding, err := h.embeddingSvc.GetTextEmbedding(req.Query)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to get embedding: "+err.Error())
		return
	}

	vectorStr := h.embeddingSvc.VectorToString(embedding)

	// Search similar images
	results, err := h.repo.SearchSimilarImages(vectorStr, 10)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to search images: "+err.Error())
		return
	}

	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			ID:         int64(r.ID),
			ImageURL:   r.ImageURL,
			Description: r.Description,
			Similarity: r.Similarity,
		}
	}

	writeJSON(w, http.StatusOK, SearchResponse{Results: searchResults})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
