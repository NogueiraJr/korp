package models

import "time"

type Product struct {
	Code          string    `json:"code"`
	Description   string    `json:"description"`
	StockQuantity int       `json:"stock_quantity"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateProductRequest struct {
	Code          string `json:"code"`
	Description   string `json:"description"`
	StockQuantity int    `json:"stock_quantity"`
}

type UpdateProductRequest struct {
	Description   string `json:"description"`
	StockQuantity int    `json:"stock_quantity"`
}

type ConsumeStockRequest struct {
	Items []ConsumeItem `json:"items"`
}

type ConsumeItem struct {
	Code     string `json:"code"`
	Quantity int    `json:"quantity"`
}

type AIDescriptionRequest struct {
	Code string `json:"code"`
}

type AIDescriptionResponse struct {
	Suggestion string `json:"suggestion"`
}
