package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"korp/faturamento/internal/models"
)

type AuthHandler struct {
	Username    string
	Password    string
	JWTSecret   string
	TokenTTL    time.Duration
	Logs        *LogsHandler
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username != h.Username || req.Password != h.Password {
		h.Logs.Log(r.Context(), "auth.login_failed", "auth", map[string]interface{}{
			"username": req.Username,
		})
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	claims := jwt.MapClaims{
		"sub": req.Username,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(h.TokenTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(h.JWTSecret))
	if err != nil {
		log.Printf("error signing token: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, models.LoginResponse{Token: signed, Username: req.Username})

	h.Logs.Log(r.Context(), "auth.login", "auth", map[string]interface{}{
		"username": req.Username,
	})
}