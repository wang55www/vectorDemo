package mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"vectorDemo/internal/repository"
	"vectorDemo/internal/service"
)

// MCPServer implements a simple MCP server using SSE
type MCPServer struct {
	repo         *repository.ImageRepository
	embeddingSvc *service.EmbeddingService
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

func NewMCPServer(repo *repository.ImageRepository, embeddingSvc *service.EmbeddingService) *MCPServer {
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
				Capabilities: map[string]interface{}{
					"tools": map[string]interface{}{},
				},
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
						Name:        "search_similar_images",
						Description: "Search for similar images by providing an image URL. Returns descriptions of the most similar images stored in the database.",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"image_url": map[string]interface{}{
									"type":        "string",
									"description": "The URL of the image to search for similar images",
								},
							},
							"required": []string{"image_url"},
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
	case "search_similar_images":
		imageURL, ok := params.Arguments["image_url"].(string)
		if !ok {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: -32602, Message: "image_url is required"},
			}
		}

		embedding, err := s.embeddingSvc.GetImageEmbedding(imageURL)
		if err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: -32603, Message: fmt.Sprintf("Failed to get embedding: %v", err)},
			}
		}

		vectorStr := s.embeddingSvc.VectorToString(embedding)
		images, err := s.repo.SearchSimilarImages(vectorStr, 5)
		if err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: -32603, Message: fmt.Sprintf("Failed to search images: %v", err)},
			}
		}

		if len(images) == 0 {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": "No similar images found in the database.",
						},
					},
				},
			}
		}

		result := "Found similar images:\n"
		for i, img := range images {
			result += fmt.Sprintf("%d. Description: %s, Image URL: %s\n", i+1, img.Description, img.ImageURL)
		}

		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": result,
					},
				},
			},
		}

	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32602, Message: "Unknown tool: " + params.Name},
		}
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