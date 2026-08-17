package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"korp/estoque/internal/ai"
	"korp/estoque/internal/models"
	"korp/estoque/internal/repository"
)

type ProductHandler struct {
	Repo     *repository.ProductRepository
	Activity *repository.ActivityRepository
	AI       *ai.Service
}

func (h *ProductHandler) logEvent(ctx context.Context, event string, details interface{}) {
	if err := h.Activity.Log(ctx, event, "product", details); err != nil {
		log.Printf("error writing activity log (%s): %v", event, err)
	}
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	req.Description = strings.TrimSpace(req.Description)
	if req.Code == "" || req.Description == "" {
		writeError(w, http.StatusBadRequest, "code and description are required")
		return
	}
	if req.StockQuantity < 0 {
		writeError(w, http.StatusBadRequest, "stock_quantity must be >= 0")
		return
	}

	product := models.Product{
		Code:          req.Code,
		Description:   req.Description,
		StockQuantity: req.StockQuantity,
	}
	if err := h.Repo.Create(r.Context(), &product); err != nil {
		if isDuplicateKeyError(err) {
			writeError(w, http.StatusConflict, "a product with this code already exists")
			return
		}
		writeInternalError(w, err)
		return
	}
	h.logEvent(r.Context(), "product.created", map[string]interface{}{
		"code": product.Code, "description": product.Description, "stock_quantity": product.StockQuantity,
	})
	writeJSON(w, http.StatusCreated, product)
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	products, err := h.Repo.List(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, products)
}

func (h *ProductHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	product, err := h.Repo.GetByCode(r.Context(), code)
	if err != nil {
		if err == repository.ErrProductNotFound {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	var req models.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Description = strings.TrimSpace(req.Description)
	if req.Description == "" || req.StockQuantity < 0 {
		writeError(w, http.StatusBadRequest, "valid description and stock_quantity are required")
		return
	}
	if err := h.Repo.Update(r.Context(), code, req.Description, req.StockQuantity); err != nil {
		if err == repository.ErrProductNotFound {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeInternalError(w, err)
		return
	}
	product, err := h.Repo.GetByCode(r.Context(), code)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	h.logEvent(r.Context(), "product.updated", map[string]interface{}{
		"code": code, "description": product.Description, "stock_quantity": product.StockQuantity,
	})
	writeJSON(w, http.StatusOK, product)
}

func (h *ProductHandler) AIDescription(w http.ResponseWriter, r *http.Request) {
	var req models.AIDescriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	suggestion, err := h.AI.SuggestDescription(r.Context(), req.Code)
	if err != nil {
		// AI failures should not break the workflow; respond with fallback
		// description and a warning flag.
		h.logEvent(r.Context(), "product.ai_description", map[string]interface{}{
			"code": req.Code, "fallback": true,
		})
		writeJSON(w, http.StatusOK, models.AIDescriptionResponse{Suggestion: suggestion})
		return
	}
	h.logEvent(r.Context(), "product.ai_description", map[string]interface{}{
		"code": req.Code, "fallback": false,
	})
	writeJSON(w, http.StatusOK, models.AIDescriptionResponse{Suggestion: suggestion})
}

func (h *ProductHandler) ConsumeStock(w http.ResponseWriter, r *http.Request) {
	var req models.ConsumeStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.Repo.ConsumeStock(r.Context(), req.Items); err != nil {
		if err == repository.ErrInsufficientStock {
			h.logEvent(r.Context(), "product.stock_insufficient", map[string]interface{}{
				"items": req.Items,
			})
			writeError(w, http.StatusConflict, "insufficient stock for one or more products")
			return
		}
		if err == repository.ErrProductNotFound {
			h.logEvent(r.Context(), "product.stock_not_found", map[string]interface{}{
				"items": req.Items,
			})
			writeError(w, http.StatusNotFound, "one or more products were not found")
			return
		}
		writeInternalError(w, err)
		return
	}
	h.logEvent(r.Context(), "product.stock_consumed", map[string]interface{}{
		"items": req.Items,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}