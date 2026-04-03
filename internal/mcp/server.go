package mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"vectorDemo/internal/model"
	"vectorDemo/internal/repository"
	"vectorDemo/internal/service"
)

// MCPServer implements MCP service with skill-based interface
type MCPServer struct {
	repo         *repository.ImageRepository
	embeddingSvc service.EmbeddingServiceInterface
	mu           sync.Mutex
	clients      map[string]bool
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolDefinition represents an MCP tool definition
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// InitializeResult represents the result of initialize method
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      map[string]string      `json:"serverInfo"`
}

func NewMCPServer(repo *repository.ImageRepository, embeddingSvc service.EmbeddingServiceInterface) *MCPServer {
	return &MCPServer{
		repo:         repo,
		embeddingSvc: embeddingSvc,
		clients:      make(map[string]bool),
	}
}

func (s *MCPServer) Start(addr string) error {
	http.HandleFunc("/sse", s.handleSSE)
	http.HandleFunc("/mcp", s.handleMCP)

	log.Printf("MCP server listening on %s", addr)
	return http.ListenAndServe(addr, nil)
}

func (s *MCPServer) Stop() error {
	return nil
}

func (s *MCPServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	clientID := r.RemoteAddr
	s.mu.Lock()
	s.clients[clientID] = true
	s.mu.Unlock()

	// Send server info
	fmt.Fprintf(w, "event: server-info\ndata: %s\n\n", jsonEncode(map[string]string{
		"name":    "Image Vector Search",
		"version": "1.0.0",
	}))
	flusher.Flush()

	// Keep connection alive
	for {
		select {
		case <-r.Context().Done():
			s.mu.Lock()
			delete(s.clients, clientID)
			s.mu.Unlock()
			return
		}
	}
}

func (s *MCPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRPCError(w, nil, -32700, "Parse error")
		return
	}

	resp := s.processRequest(&req)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *MCPServer) processRequest(req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: InitializeResult{
				ProtocolVersion: "2024-11-05",
				Capabilities: map[string]interface{}{},
				ServerInfo: map[string]string{
					"name":    "Image Vector Search",
					"version": "1.0.0",
				},
			},
		}

	case "tools/list":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": []ToolDefinition{
					{
						Name:        "save_image",
						Description: "保存图片到平凯数据库。当用户提供了图片地址并说'保存到平凯数据库'时使用此工具。",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"image_url": map[string]interface{}{
									"type":        "string",
									"description": "图片的URL地址",
								},
								"description": map[string]interface{}{
									"type":        "string",
									"description": "图片描述信息",
								},
							},
							"required": []string{"image_url"},
						},
					},
					{
						Name:        "search_images",
						Description: "通过文字描述搜索相似图片。当用户说'我要找一张xx的照片'时使用此工具，将xx作为查询文本。",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"query": map[string]interface{}{
									"type":        "string",
									"description": "搜索关键词，如'风景'、'动物'等",
								},
							},
							"required": []string{"query"},
						},
					},
				},
			},
		}

	case "tools/call":
		return s.handleToolCall(req)

	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32601, Message: "Method not found"},
		}
	}
}

func (s *MCPServer) handleToolCall(req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32602, Message: "Invalid params"},
		}
	}

	switch params.Name {
	case "save_image":
		return s.handleSaveImage(req.ID, params.Arguments)
	case "search_images":
		return s.handleSearchImages(req.ID, params.Arguments)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32602, Message: "Unknown tool: " + params.Name},
		}
	}
}

// handleSaveImage 处理保存图片到平凯数据库
func (s *MCPServer) handleSaveImage(id interface{}, args map[string]interface{}) *JSONRPCResponse {
	imageURL, ok := args["image_url"].(string)
	if !ok || imageURL == "" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &RPCError{Code: -32602, Message: "image_url is required"},
		}
	}

	description := ""
	if desc, ok := args["description"].(string); ok {
		description = desc
	}

	// Get image embedding from Jina API
	embedding, err := s.embeddingSvc.GetImageEmbedding(imageURL)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &RPCError{Code: -32603, Message: fmt.Sprintf("Failed to get embedding: %v", err)},
		}
	}

	vectorStr := s.embeddingSvc.VectorToString(embedding)

	// Insert into database
	image := &model.Image{
		ImageURL:    imageURL,
		Description: description,
		Vector:      vectorStr,
	}

	_, err = s.repo.InsertImage(image)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &RPCError{Code: -32603, Message: fmt.Sprintf("Failed to save image: %v", err)},
		}
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("图片已成功保存到平凯数据库！\n图片地址: %s\n描述: %s", imageURL, description),
				},
			},
		},
	}
}

// handleSearchImages 处理文字搜索图片
func (s *MCPServer) handleSearchImages(id interface{}, args map[string]interface{}) *JSONRPCResponse {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &RPCError{Code: -32602, Message: "query is required"},
		}
	}

	// Get text embedding from Jina API
	embedding, err := s.embeddingSvc.GetTextEmbedding(query)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &RPCError{Code: -32603, Message: fmt.Sprintf("Failed to get embedding: %v", err)},
		}
	}

	vectorStr := s.embeddingSvc.VectorToString(embedding)
	searchResults, err := s.repo.SearchSimilarImages(vectorStr, 5)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &RPCError{Code: -32603, Message: fmt.Sprintf("Failed to search images: %v", err)},
		}
	}

	if len(searchResults) == 0 {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": fmt.Sprintf("未找到与'%s'相似的图片。", query),
					},
				},
			},
		}
	}

	result := fmt.Sprintf("为您找到 %d 张与'%s'相似的图片：\n\n", len(searchResults), query)
	for i, r := range searchResults {
		result += fmt.Sprintf("%d. 描述: %s\n   图片地址: %s\n   相似度: %.4f\n\n", i+1, r.Description, r.ImageURL, r.Similarity)
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": result,
				},
			},
		},
	}
}

func writeRPCError(w http.ResponseWriter, id interface{}, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: message},
	})
}

func jsonEncode(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}