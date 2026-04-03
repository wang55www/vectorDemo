package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"vectorDemo/internal/config"
)

type EmbeddingService struct {
	jinaConfig  *config.JinaConfig
	ollamaURL   string
	useOllama   bool
}

func NewEmbeddingService(jinaConfig *config.JinaConfig) *EmbeddingService {
	// 检查是否使用 Ollama（本地优先）
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	
	// 测试 Ollama 是否可用
	useOllama := true
	resp, err := http.Get(ollamaURL + "/api/tags")
	if err != nil || resp.StatusCode != 200 {
		useOllama = false
	}
	
	return &EmbeddingService{
		jinaConfig: jinaConfig,
		ollamaURL:  ollamaURL,
		useOllama:  useOllama,
	}
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

// GetImageEmbedding 获取图片向量（支持URL或base64）
func (s *EmbeddingService) GetImageEmbedding(imageURL string) ([]float64, error) {
	reqBody := JinaEmbeddingRequest{
		Model: "jina-clip-v2",
		Input: []JinaInput{
			{Image: imageURL},
		},
	}

	return s.callJinaAPI(reqBody)
}

// GetTextEmbedding 获取文字向量
func (s *EmbeddingService) GetTextEmbedding(text string) ([]float64, error) {
	reqBody := JinaEmbeddingRequest{
		Model: "jina-clip-v2",
		Input: []JinaInput{
			{Text: text},
		},
	}

	return s.callJinaAPI(reqBody)
}

// GetImageEmbeddingFromBase64 获取本地图片base64向量
func (s *EmbeddingService) GetImageEmbeddingFromBase64(base64Data string) ([]float64, error) {
	// base64 格式: data:image/jpeg;base64,/9j/4AAQ...
	reqBody := JinaEmbeddingRequest{
		Model: "jina-clip-v2",
		Input: []JinaInput{
			{Image: base64Data},
		},
	}

	return s.callJinaAPI(reqBody)
}

func (s *EmbeddingService) callJinaAPI(reqBody JinaEmbeddingRequest) ([]float64, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", s.jinaConfig.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.jinaConfig.APIKey))

	// 创建带代理和超时的客户端
	transport := &http.Transport{}

	// 如果配置了代理，使用代理
	if s.jinaConfig.Proxy != "" {
		proxyURL, err := url.Parse(s.jinaConfig.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}

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

// FileToBase64 将文件内容转为base64编码
func (s *EmbeddingService) FileToBase64(data []byte, contentType string) string {
	base64Str := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", contentType, base64Str)
}

// GetImageEmbeddingFromFile 获取本地图片文件的向量（Jina 不支持，返回错误）
func (s *EmbeddingService) GetImageEmbeddingFromFile(filePath string) ([]float64, error) {
	return nil, fmt.Errorf("Jina API 不支持本地图片文件嵌入")
}

// GetImageEmbeddingFromFilePath 获取本地图片文件的向量（从文件路径）
func (s *EmbeddingService) GetImageEmbeddingFromFilePath(filePath string) ([]float64, error) {
	return nil, fmt.Errorf("Jina API 不支持本地图片文件嵌入")
}