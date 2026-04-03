package model

import "time"

type Image struct {
	ID          int       `db:"id" json:"id"`
	ImageURL    string    `db:"image_url" json:"image_url"`
	Description string    `db:"description" json:"description"`
	Vector      string    `db:"vector" json:"vector"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// UploadImageRequest 支持URL或本地文件上传
type UploadImageRequest struct {
	ImageURL    string `form:"image_url" json:"image_url"`
	Description string `form:"description" json:"description"`
}

// SearchImageRequest 文字搜索请求
type SearchImageRequest struct {
	Query string `form:"query" json:"query"`
}

// SearchImageResponse 图片搜索结果
type SearchImageResponse struct {
	ID          int     `json:"id"`
	Description string  `json:"description"`
	ImageURL    string  `json:"image_url"`
	Similarity  float64 `json:"similarity"`
}