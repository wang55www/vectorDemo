# 图片向量搜索服务

一个基于 Go 语言实现的图片向量搜索服务，使用 TiDB 存储图片向量，通过 Jina API 进行图片和文字向量化，支持 HTTP 接口和 MCP Skill 接口。

## 功能特性

- 图片上传及向量存储（支持URL和本地文件）
- 通过文字描述搜索相似图片
- HTTP REST API 接口
- MCP Skill 接口，支持自然语言交互

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

支持两种方式：**URL上传** 或 **本地文件上传**

#### 方式一：通过 URL 上传（JSON 格式）

**接口**: `POST /api/images`

**Content-Type**: `application/json`

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

#### 方式二：上传本地文件（FormData 格式）

**接口**: `POST /api/images`

**Content-Type**: `multipart/form-data`

**请求参数**:
- `image`: 图片文件 (必填)
- `description`: 图片描述 (可选)

**示例**:

```bash
curl -X POST http://localhost:8080/api/images \
  -F "image=@/path/to/local/image.jpg" \
  -F "description=这是一张本地图片"
```

**响应**:

```json
{
  "id": 2,
  "image_url": "file://image.jpg",
  "description": "这是一张本地图片"
}
```

### 2. 通过文字搜索相似图片

**接口**: `POST /api/images/search`

**请求参数**:
- `query`: 搜索关键词/描述 (必填)

**示例**:

```bash
curl -X POST http://localhost:8080/api/images/search \
  -H "Content-Type: application/json" \
  -d '{"query": "风景照片"}'
```

**响应**:

```json
{
  "results": [
    {
      "id": 1,
      "description": "这是一张风景图片",
      "image_url": "https://example.com/image.jpg"
    },
    {
      "id": 2,
      "description": "另一张风景图",
      "image_url": "https://example.com/image2.jpg"
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

## MCP Skill 接口说明

MCP 服务运行在端口 8081，提供两个 Skill 工具供 AI Agent 使用：

### 1. save_image - 保存图片到平凯数据库

**触发条件**: 当用户提供了图片地址并说"保存到平凯数据库"时

**参数**:
- `image_url` (string, 必填): 图片 URL
- `description` (string, 可选): 图片描述

**使用示例**:

用户说："https://example.com/beauty.jpg 保存到平凯数据库"

Agent 调用：
```json
{
  "name": "save_image",
  "arguments": {
    "image_url": "https://example.com/beauty.jpg",
    "description": ""
  }
}
```

### 2. search_images - 通过文字描述搜索图片

**触发条件**: 当用户说"我要找一张xx的照片"时，将xx作为查询文本

**参数**:
- `query` (string, 必填): 搜索关键词

**使用示例**:

用户说："我要找一张风景的照片"

Agent 调用：
```json
{
  "name": "search_images",
  "arguments": {
    "query": "风景"
  }
}
```

### MCP 配置示例

在 Claude Code 或其他 MCP 客户端配置：

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
- TiDB 地址: 127.0.0.1:4000
- 用户名: root
- 密码: 空
- 数据库: vector_demo

### 数据表结构

```sql
CREATE TABLE IF NOT EXISTS images (
    id INT AUTO_INCREMENT PRIMARY KEY,
    image_url VARCHAR(1024) NOT NULL,
    description TEXT,
    vector VECTOR(1024),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 向量模型

使用 Jina CLIP v2 模型进行向量嵌入：
- API: https://api.jina.ai/v1/embeddings
- 向量维度: 1024
- 距离计算: 余弦距离 (Cosine Distance)
- 支持：图片向量化、文字向量化

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
│   └── mcp/server.go        # MCP Skill 服务
├── go.mod                   # Go 模块定义
├── go.sum                   # 依赖版本锁定
└── README.md                # 项目说明
```

## 依赖包

- [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) v1.7.1 - MySQL/TiDB 驱动
- 标准库 net/http 实现 HTTP 服务和 MCP 服务