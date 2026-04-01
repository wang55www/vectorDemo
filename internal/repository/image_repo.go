package repository

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"

	"vectorDemo/internal/config"
	"vectorDemo/internal/model"
)

type ImageRepository struct {
	db *sql.DB
}

func NewImageRepository(cfg *config.TiDBConfig) (*ImageRepository, error) {
	// First connect without specifying database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?parseTime=true&multiStatements=true",
		cfg.User, cfg.Password, cfg.Host, cfg.Port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &ImageRepository{db: db}, nil
}

func (r *ImageRepository) InitSchema() error {
	// Create database if not exists
	_, err := r.db.Exec("CREATE DATABASE IF NOT EXISTS vector_demo")
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	_, err = r.db.Exec("USE vector_demo")
	if err != nil {
		return fmt.Errorf("failed to use database: %w", err)
	}

	// Create images table with vector column
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS images (
		id INT AUTO_INCREMENT PRIMARY KEY,
		image_url VARCHAR(1024) NOT NULL,
		description TEXT,
		vector VECTOR(1024),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err = r.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

func (r *ImageRepository) InsertImage(image *model.Image) (int64, error) {
	query := "INSERT INTO images (image_url, description, vector) VALUES (?, ?, ?)"

	var vectorStr interface{}
	if image.Vector != "" {
		vectorStr = image.Vector
	} else {
		vectorStr = nil
	}

	result, err := r.db.Exec(query, image.ImageURL, image.Description, vectorStr)
	if err != nil {
		return 0, fmt.Errorf("failed to insert image: %w", err)
	}

	return result.LastInsertId()
}

func (r *ImageRepository) SearchSimilarImages(vectorStr string, limit int) ([]model.Image, error) {
	// Use TiDB's vector distance function (cosine distance)
	query := fmt.Sprintf(`
		SELECT id, image_url, description, created_at,
			   VEC_Cosine_Distance(vector, '%s') as similarity
		FROM images
		WHERE vector IS NOT NULL
		ORDER BY similarity ASC
		LIMIT ?
	`, vectorStr)

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search images: %w", err)
	}
	defer rows.Close()

	var images []model.Image
	for rows.Next() {
		var img model.Image
		var similarity float64
		if err := rows.Scan(&img.ID, &img.ImageURL, &img.Description, &img.CreatedAt, &similarity); err != nil {
			return nil, fmt.Errorf("failed to scan image: %w", err)
		}
		images = append(images, img)
	}

	return images, nil
}

func (r *ImageRepository) Close() error {
	return r.db.Close()
}