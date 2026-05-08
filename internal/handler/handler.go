package handler

//go:generate go tool mockgen -destination=./mocks/auth_service_mock.go -package=mocks . AuthServiceInterface

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"go-musthave-diploma-tpl/internal/dto"
	"go-musthave-diploma-tpl/internal/service"
	myErrors "go-musthave-diploma-tpl/pkg/errors"
)

type Handler struct {
	authService *service.AuthService
}

type AuthServiceInterface interface {
	Register(login, password string) (string, error)
	Login(login, password string) (string, error)
	GetUserIDFromToken(token string) (int64, error)
	UploadOrder(userID int64, orderNumber string) (int, error)
	GetOrders(userID int64) ([]dto.Order, error)
	GetBalance(userID int64) (*dto.BalanceResponse, error)
	Withdraw(userID int64, orderNumber string, sum float64) error
	GetWithdrawals(userID int64) ([]dto.Withdrawal, error)
}

func NewHandler(authService *service.AuthService) *Handler {
	return &Handler{authService: authService}
}

func errorJSON(err error) string {
	return `{"error": "` + err.Error() + `"}`
}

func encode(w http.ResponseWriter, v interface{}) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, errorJSON(myErrors.ErrInternalError), http.StatusInternalServerError)
	}
}

func decode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, errorJSON(myErrors.ErrInvalidBody), http.StatusBadRequest)
		return
	}
	var req dto.RegisterRequest
	if err := decode(body, &req); err != nil {
		http.Error(w, errorJSON(myErrors.ErrInvalidBody), http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, `{"error": "требуются логин и пароль"}`, http.StatusBadRequest)
		return
	}

	token, err := h.authService.Register(req.Login, req.Password)
	if err != nil {
		if errors.Is(err, myErrors.ErrAlreadyExists) {
			http.Error(w, `{"error": "логин уже существует"}`, http.StatusConflict)
			return
		}
		http.Error(w, errorJSON(myErrors.ErrInternalError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encode(w, dto.RegisterResponse{Token: token})
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, errorJSON(myErrors.ErrInvalidBody), http.StatusBadRequest)
		return
	}
	var req dto.LoginRequest
	if err := decode(body, &req); err != nil {
		http.Error(w, errorJSON(myErrors.ErrInvalidBody), http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, `{"error": "требуются логин и пароль"}`, http.StatusBadRequest)
		return
	}

	token, err := h.authService.Login(req.Login, req.Password)
	if err != nil {
		if errors.Is(err, myErrors.ErrInvalidCreds) {
			http.Error(w, `{"error": "некорректные учётные данные"}`, http.StatusUnauthorized)
			return
		}
		http.Error(w, errorJSON(myErrors.ErrInternalError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encode(w, dto.LoginResponse{Token: token})
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		http.Error(w, `{"error": "требуется авторизация"}`, http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, errorJSON(myErrors.ErrInvalidBody), http.StatusBadRequest)
		return
	}
	orderNumber := strings.TrimSpace(string(body))
	if orderNumber == "" {
		http.Error(w, `{"error": "некорректный номер заказа"}`, http.StatusBadRequest)
		return
	}

	statusCode, err := h.authService.UploadOrder(userID, orderNumber)
	if err != nil {
		if errors.Is(err, myErrors.ErrInvalidOrder) {
			http.Error(w, `{"error": "некорректный номер заказа"}`, http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, myErrors.ErrAlreadyExists) {
			http.Error(w, `{"error": "заказ уже существует"}`, http.StatusConflict)
			return
		}
		http.Error(w, errorJSON(myErrors.ErrInternalError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(statusCode)
}

func (h *Handler) GetOrders(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		http.Error(w, `{"error": "требуется авторизация"}`, http.StatusUnauthorized)
		return
	}

	orders, err := h.authService.GetOrders(userID)
	if err != nil {
		http.Error(w, errorJSON(myErrors.ErrInternalError), http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encode(w, orders)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		http.Error(w, `{"error": "требуется авторизация"}`, http.StatusUnauthorized)
		return
	}

	balance, err := h.authService.GetBalance(userID)
	if err != nil {
		http.Error(w, errorJSON(myErrors.ErrInternalError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encode(w, balance)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		http.Error(w, `{"error": "требуется авторизация"}`, http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, errorJSON(myErrors.ErrInvalidBody), http.StatusBadRequest)
		return
	}
	var req dto.WithdrawRequest
	if err := decode(body, &req); err != nil {
		http.Error(w, errorJSON(myErrors.ErrInvalidBody), http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, `{"error": "некорректный запрос"}`, http.StatusBadRequest)
		return
	}

	err = h.authService.Withdraw(userID, req.Order, req.Sum)
	if err != nil {
		if errors.Is(err, myErrors.ErrInvalidBody) {
			http.Error(w, errorJSON(myErrors.ErrInvalidBody), http.StatusBadRequest)
			return
		}
		if errors.Is(err, myErrors.ErrInsufficientFunds) {
			http.Error(w, `{"error": "недостаточно средств"}`, http.StatusPaymentRequired)
			return
		}
		if errors.Is(err, myErrors.ErrInvalidOrder) {
			http.Error(w, `{"error": "некорректный номер заказа"}`, http.StatusUnprocessableEntity)
			return
		}
		http.Error(w, errorJSON(myErrors.ErrInternalError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		http.Error(w, `{"error": "требуется авторизация"}`, http.StatusUnauthorized)
		return
	}

	withdrawals, err := h.authService.GetWithdrawals(userID)
	if err != nil {
		http.Error(w, errorJSON(myErrors.ErrInternalError), http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encode(w, withdrawals)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) getUserID(r *http.Request) (int64, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return 0, myErrors.ErrUnauthorized
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return 0, myErrors.ErrUnauthorized
	}
	return h.authService.GetUserIDFromToken(parts[1])
}
