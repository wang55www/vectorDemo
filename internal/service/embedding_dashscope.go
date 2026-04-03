package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"vectorDemo/internal/config"
)

// EmbeddingServiceInterface 嵌入服务接口
type EmbeddingServiceInterface interface {
	GetTextEmbedding(text string) ([]float64, error)
	GetImageEmbedding(imageURL string) ([]float64, error)
	GetImageEmbeddingFromBase64(base64Data string) ([]float64, error)
	GetImageEmbeddingFromFile(filePath string) ([]float64, error)
	GetImageEmbeddingFromFilePath(filePath string) ([]float64, error)
	FileToBase64(data []byte, contentType string) string
	VectorToString(embedding []float64) string
}

// DashScope 多模态请求结构
type MultimodalRequest struct {
	Model      string                 `json:"model"`
	Input      map[string]interface{} `json:"input"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// DashScope 多模态响应结构
type MultimodalResponse struct {
	Output struct {
		Embeddings []struct {
			Index     int       `json:"text_index"`
			Embedding []float64 `json:"embedding"`
		} `json:"embeddings"`
	} `json:"output"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}

// DashScopeService 阿里云 DashScope 多模态嵌入服务
type DashScopeService struct {
	APIKey          string
	APIURL          string
	Model           string
	VectorDimension int
}

func NewDashScopeService(apiKey, apiURL, model string) *DashScopeService {
	return &DashScopeService{
		APIKey:          apiKey,
		APIURL:          apiURL,
		Model:           model,
		VectorDimension: 2560, // qwen3-vl-embedding 默认 2560 维
	}
}

// NewDashScopeServiceWithConfig 使用配置创建服务
func NewDashScopeServiceWithConfig(cfg *config.DashScopeConfig) *DashScopeService {
	dimension := 2560
	if cfg.VectorDimension > 0 {
		dimension = cfg.VectorDimension
	}
	
	return &DashScopeService{
		APIKey:          cfg.APIKey,
		APIURL:          cfg.APIURL,
		Model:           cfg.Model,
		VectorDimension: dimension,
	}
}

// GetTextEmbedding 获取文本向量
func (s *DashScopeService) GetTextEmbedding(text string) ([]float64, error) {
	reqBody := MultimodalRequest{
		Model: s.Model,
		Input: map[string]interface{}{
			"contents": []map[string]interface{}{
				{"text": text},
			},
		},
	}
	
	return s.callAPI(reqBody)
}

// GetImageEmbedding 获取图片向量（通过 URL）
func (s *DashScopeService) GetImageEmbedding(imageURL string) ([]float64, error) {
	// 如果是本地文件 URL，读取文件内容
	if len(imageURL) > 0 && imageURL[:1] == "/" {
		return s.GetImageEmbeddingFromFile(imageURL)
	}
	
	// 使用 contents 字段
	reqBody := MultimodalRequest{
		Model: s.Model,
		Input: map[string]interface{}{
			"contents": []map[string]interface{}{
				{"image": imageURL},
			},
		},
	}
	
	return s.callAPI(reqBody)
}

// GetImageEmbeddingFromBase64 获取图片向量（通过 base64）
func (s *DashScopeService) GetImageEmbeddingFromBase64(base64Data string) ([]float64, error) {
	reqBody := MultimodalRequest{
		Model: s.Model,
		Input: map[string]interface{}{
			"contents": []map[string]interface{}{
				{"image": base64Data},
			},
		},
	}
	
	return s.callAPI(reqBody)
}

// GetImageEmbeddingFromFile 获取图片向量（通过本地文件路径）
func (s *DashScopeService) GetImageEmbeddingFromFile(filePath string) ([]float64, error) {
	// 读取文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}
	
	// 转换为 base64
	base64Data := base64.StdEncoding.EncodeToString(data)
	
	return s.GetImageEmbeddingFromBase64(base64Data)
}

// GetImageEmbeddingFromFilePath 从本地文件路径读取图片并获取向量
func (s *DashScopeService) GetImageEmbeddingFromFilePath(filePath string) ([]float64, error) {
	// 读取文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}
	
	// 转换为 base64
	base64Data := base64.StdEncoding.EncodeToString(data)
	
	// 转为 data URL 格式
	// 根据文件扩展名判断 content type
	contentType := "image/jpeg"
	if len(filePath) > 4 {
		ext := filePath[len(filePath)-4:]
		switch ext {
		case ".png":
			contentType = "image/png"
		case ".gif":
			contentType = "image/gif"
		case ".webp":
			contentType = "image/webp"
		}
	}
	dataURL := fmt.Sprintf("data:%s;base64,%s", contentType, base64Data)
	
	return s.GetImageEmbeddingFromBase64(dataURL)
}

func (s *DashScopeService) callAPI(reqBody MultimodalRequest) ([]float64, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", s.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.APIKey))

	client := &http.Client{
		Timeout: 120 * time.Second,
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

	var respData MultimodalResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(respData.Output.Embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned from API")
	}

	return respData.Output.Embeddings[0].Embedding, nil
}

// FileToBase64 将文件内容转为 base64 编码
func (s *DashScopeService) FileToBase64(data []byte, contentType string) string {
	return fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(data))
}

// VectorToString 向量转字符串
func (s *DashScopeService) VectorToString(embedding []float64) string {
	jsonBytes, _ := json.Marshal(embedding)
	return string(jsonBytes)
}