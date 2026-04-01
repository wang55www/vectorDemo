# 图片向量搜索服务

一个基于 Go 语言实现的图片向量搜索服务，使用 TiDB 存储图片向量，通过 Jina API 进行图片向量化，支持 HTTP 和 MCP 接口。

## 功能特性

- 图片上传及向量存储
- 基于向量相似度的图片搜索
- HTTP REST API 接口
- MCP (Model Context Protocol) 接口，供 AI Agent 调用

## 系统架构

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   HTTP Client   │     │    MCP Agent    │     │   Jina API      │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │                       │
        └───────────────────────┼───────────────────────┘
                                │
                        ┌───────┴───────┐
                        │   Go Server   │
                        │  (HTTP + MCP) │
                        └───────┬───────┘
                                │
                        ┌───────┴───────┐
                        │     TiDB      │
                        │  (Vector DB)  │
                        └───────────────┘
```

## 环境要求

- Go 1.18+
- TiDB (支持向量类型)
- 网络访问 Jina API

## 安装与运行

### 1. 克隆项目

```bash
cd vectorDemo
```

### 2. 安装依赖

```bash
go mod tidy
```

### 3. 运行服务

```bash
go run cmd/server/main.go
```

服务启动后：
- HTTP 服务监听端口: 8080
- MCP 服务监听端口: 8081

## HTTP 接口说明

### 1. 上传图片

**接口**: `POST /api/images`

**请求参数**:
- `image_url`: 图片 URL (必填)
- `description`: 图片描述 (可选)

**示例**:

```bash
curl -X POST http://localhost:8080/api/images \
  -H "Content-Type: application/json" \
  -d '{"image_url": "https://example.com/image.jpg", "description": "这是一张风景图片"}'
```

**响应**:

```json
{
  "id": 1,
  "image_url": "https://example.com/image.jpg",
  "description": "这是一张风景图片"
}
```

### 2. 搜索相似图片

**接口**: `POST /api/images/search`

**请求参数**:
- `image_url`: 要搜索的图片 URL (必填)

**示例**:

```bash
curl -X POST http://localhost:8080/api/images/search \
  -H "Content-Type: application/json" \
  -d '{"image_url": "https://example.com/query-image.jpg"}'
```

**响应**:

```json
{
  "results": [
    {
      "id": 1,
      "description": "这是一张风景图片",
      "image_url": "https://example.com/image.jpg",
      "similarity": "0.15"
    },
    {
      "id": 2,
      "description": "另一张风景图",
      "image_url": "https://example.com/image2.jpg",
      "similarity": "0.23"
    }
  ]
}
```

### 3. 健康检查

**接口**: `GET /health`

**响应**:

```json
{
  "status": "ok"
}
```

## MCP 接口说明

MCP 服务运行在端口 8081，提供以下工具供 AI Agent 使用：

### search_similar_images 工具

**功能**: 通过图片 URL 搜索数据库中相似的图片

**参数**:
- `image_url` (string, 必填): 图片 URL

**返回**: 最相似图片的描述信息列表

**使用示例**:

在 Claude Code 中配置 MCP 服务：

```json
{
  "mcpServers": {
    "image-search": {
      "url": "http://localhost:8081/sse"
    }
  }
}
```

## 数据库配置

默认配置：
- TiDB 地址: 192.168.1.13:4000
- 用户名: root
- 密码: 空
- 数据库: vector_demo

## 向量模型

使用 Jina CLIP v2 模型进行图片向量嵌入：
- API: https://api.jina.ai/v1/embeddings
- 向量维度: 1024
- 距离计算: 余弦距离 (Cosine Distance)

## 项目结构

```
vectorDemo/
├── cmd/server/main.go       # 主程序入口
├── internal/
│   ├── config/config.go     # 配置管理
│   ├── handler/handler.go   # HTTP 处理器
│   ├── model/image.go       # 数据模型
│   ├── repository/image_repo.go  # 数据库操作
│   ├── service/embedding.go # 向量嵌入服务
│   └── mcp/server.go        # MCP 服务
├── go.mod                   # Go 模块定义
├── go.sum                   # 依赖版本锁定
└── README.md                # 项目说明
```

## 依赖包

- [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) v1.7.1 - MySQL/TiDB 驱动
- 标准库 net/http 实现 HTTP 服务和 MCP 服务