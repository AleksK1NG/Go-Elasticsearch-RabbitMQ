package domain

import "time"

type Product struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	ImageURL     string    `json:"image_url"`
	CountInStock int64     `json:"count_in_stock"`
	Shop         string    `json:"shop"`
	CreatedAt    time.Time `json:"created_at"`
}
