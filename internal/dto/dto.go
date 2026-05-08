package dto

import "errors"

type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (r *RegisterRequest) Validate() error {
	if r.Login == "" {
		return errors.New("требуется логин")
	}
	if r.Password == "" {
		return errors.New("требуется пароль")
	}
	return nil
}

type RegisterResponse struct {
	Token string `json:"token"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() error {
	if r.Login == "" {
		return errors.New("требуется логин")
	}
	if r.Password == "" {
		return errors.New("требуется пароль")
	}
	return nil
}

type LoginResponse struct {
	Token string `json:"token"`
}

type Order struct {
	Number     string  `json:"number"`
	Status     string  `json:"status"`
	Accrual    float64 `json:"accrual,omitempty"`
	UploadedAt string  `json:"uploaded_at"`
}

type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type WithdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

func (r *WithdrawRequest) Validate() error {
	if r.Order == "" {
		return errors.New("требуется номер заказа")
	}
	if r.Sum <= 0 {
		return errors.New("сумма должна быть положительной")
	}
	return nil
}

type Withdrawal struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

type AccrualResponse struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}
