package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"korp/faturamento/internal/client"
	"korp/faturamento/internal/models"
	"korp/faturamento/internal/repository"
)

type InvoiceHandler struct {
	Repo    *repository.InvoiceRepository
	Estoque *client.EstoqueClient
	Logs    *LogsHandler
}

func (h *InvoiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Items) == 0 {
		writeError(w, http.StatusBadRequest, "an invoice must have at least one item")
		return
	}
	seen := map[string]bool{}
	for _, item := range req.Items {
		item.ProductCode = strings.TrimSpace(item.ProductCode)
		if item.ProductCode == "" || item.Quantity <= 0 {
			writeError(w, http.StatusBadRequest, "each item needs a product_code and quantity > 0")
			return
		}
		if seen[item.ProductCode] {
			writeError(w, http.StatusBadRequest, "duplicate product in invoice items")
			return
		}
		seen[item.ProductCode] = true
	}

	invoice, err := h.Repo.Create(r.Context(), req)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	h.Logs.Log(r.Context(), "invoice.created", "invoice", map[string]interface{}{
		"number": invoice.Number,
		"items":  invoice.Items,
	})
	writeJSON(w, http.StatusCreated, invoice)
}

func (h *InvoiceHandler) List(w http.ResponseWriter, r *http.Request) {
	invoices, err := h.Repo.List(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, invoices)
}

func (h *InvoiceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	invoice, err := h.Repo.GetByID(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			writeError(w, http.StatusNotFound, "invoice not found")
			return
		}
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, invoice)
}

// Print closes an OPEN invoice and consumes stock from the Estoque service.
// It is idempotent when the caller sends an Idempotency-Key header: repeated
// calls with the same key return the stored result without side effects.
func (h *InvoiceHandler) Print(w http.ResponseWriter, r *http.Request) {
	invoiceID := r.PathValue("id")
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))

	if key != "" {
		if stored, found, err := h.Repo.GetIdempotencyResponse(r.Context(), key); err != nil {
			writeInternalError(w, err)
			return
		} else if found {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(stored)
			return
		}
	}

	invoice, err := h.Repo.GetByID(r.Context(), invoiceID)
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			writeError(w, http.StatusNotFound, "invoice not found")
			return
		}
		writeInternalError(w, err)
		return
	}

	if invoice.Status != "OPEN" {
		h.Logs.Log(r.Context(), "invoice.print_rejected", "invoice", map[string]interface{}{
			"number": invoice.Number, "status": invoice.Status,
		})
		writeError(w, http.StatusConflict, "only OPEN invoices can be printed")
		return
	}

	items := make([]client.ConsumeItem, 0, len(invoice.Items))
	for _, item := range invoice.Items {
		items = append(items, client.ConsumeItem{Code: item.ProductCode, Quantity: item.Quantity})
	}

	if err := h.Estoque.ConsumeStock(r.Context(), items); err != nil {
		switch {
		case errors.Is(err, client.ErrInsufficientStock):
			h.Logs.Log(r.Context(), "invoice.print_error", "invoice", map[string]interface{}{
				"number": invoice.Number, "reason": "insufficient_stock",
			})
			writeError(w, http.StatusConflict, "insufficient stock for one or more products")
		case errors.Is(err, client.ErrProductNotFound):
			h.Logs.Log(r.Context(), "invoice.print_error", "invoice", map[string]interface{}{
				"number": invoice.Number, "reason": "product_not_found",
			})
			writeError(w, http.StatusConflict, "one or more products were not found in the inventory")
		default:
			// Estoque unavailable (even after retries / circuit breaker):
			// the invoice stays OPEN so the user can retry later.
			h.Logs.Log(r.Context(), "invoice.print_error", "invoice", map[string]interface{}{
				"number": invoice.Number, "reason": "estoque_unavailable",
			})
			writeError(w, http.StatusServiceUnavailable,
				"inventory service unavailable. The invoice was not closed; please try again")
		}
		return
	}

	if err := h.Repo.ClaimClose(r.Context(), invoiceID); err != nil {
		if errors.Is(err, repository.ErrInvoiceClosed) {
			writeError(w, http.StatusConflict, "invoice is already closed")
			return
		}
		writeInternalError(w, err)
		return
	}

	updated, err := h.Repo.GetByID(r.Context(), invoiceID)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	if key != "" {
		body, err := json.Marshal(updated)
		if err == nil {
			if err := h.Repo.SaveIdempotencyResponse(r.Context(), key, body); err != nil {
				// Non-fatal: the operation already succeeded.
				log.Printf("error saving idempotency response: %v", err)
			}
		}
	}

	h.Logs.Log(r.Context(), "invoice.print_success", "invoice", map[string]interface{}{
		"number": updated.Number,
		"items":  updated.Items,
	})
	writeJSON(w, http.StatusOK, updated)
}