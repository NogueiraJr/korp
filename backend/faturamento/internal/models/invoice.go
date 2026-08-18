package models

import "time"

type InvoiceItem struct {
	ID          string `json:"id"`
	ProductCode string `json:"product_code"`
	Quantity    int    `json:"quantity"`
}

type Invoice struct {
	ID        string        `json:"id"`
	Number    int           `json:"number"`
	Status    string        `json:"status"`
	Items     []InvoiceItem `json:"items"`
	CreatedAt time.Time     `json:"created_at"`
	ClosedAt  *time.Time    `json:"closed_at,omitempty"`
}

type CreateInvoiceRequest struct {
	Items []CreateInvoiceItem `json:"items"`
}

type CreateInvoiceItem struct {
	ProductCode string `json:"product_code"`
	Quantity    int    `json:"quantity"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}