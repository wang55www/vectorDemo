package model

import "time"

type Image struct {
	ID          int       `db:"id" json:"id"`
	ImageURL    string    `db:"image_url" json:"image_url"`
	Description string    `db:"description" json:"description"`
	Vector      string    `db:"vector" json:"vector"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type UploadImageRequest struct {
	ImageURL    string `form:"image_url" json:"image_url"`
	Description string `form:"description" json:"description"`
}

type SearchImageRequest struct {
	ImageURL string `form:"image_url" json:"image_url"`
}

type SearchImageResponse struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	Similarity  string `json:"similarity"`
}