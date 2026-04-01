package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"vectorDemo/internal/config"
)

type EmbeddingService struct {
	config *config.JinaConfig
}

func NewEmbeddingService(config *config.JinaConfig) *EmbeddingService {
	return &EmbeddingService{config: config}
}

type JinaEmbeddingRequest struct {
	Model string      `json:"model"`
	Input []JinaInput `json:"input"`
}

type JinaInput struct {
	Image string `json:"image,omitempty"`
	Text  string `json:"text,omitempty"`
}

type JinaEmbeddingResponse struct {
	Model string `json:"model"`
	Data  []struct {
		Index     int       `json:"index"`
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func (s *EmbeddingService) GetImageEmbedding(imageURL string) ([]float64, error) {
	reqBody := JinaEmbeddingRequest{
		Model: "jina-clip-v2",
		Input: []JinaInput{
			{Image: imageURL},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", s.config.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.APIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var jinaResp JinaEmbeddingResponse
	if err := json.Unmarshal(body, &jinaResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(jinaResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned from API")
	}

	return jinaResp.Data[0].Embedding, nil
}

func (s *EmbeddingService) VectorToString(embedding []float64) string {
	jsonBytes, _ := json.Marshal(embedding)
	return string(jsonBytes)
}