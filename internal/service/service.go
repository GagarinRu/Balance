package service

//go:generate go run go.uber.org/mock/mockgen -destination=internal/service/mocks/user_repo_mock.go -package=mocks -mock_names UserRepository,OrderRepository,AccrualClient . UserRepository OrderRepository AccrualClient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"go-musthave-diploma-tpl/internal/dto"
	"go-musthave-diploma-tpl/internal/entity"
	myErrors "go-musthave-diploma-tpl/pkg/errors"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
)

type UserRepository interface {
	CreateUser(login, passwordHash string) (int64, error)
	GetUserByLogin(login string) (*entity.User, error)
	GetUserBalance(userID int64) (float64, float64, error)
	UpdateUserBalance(userID int64, amount float64) error
	ProcessWithdrawal(userID int64, orderNumber string, sum float64) error
	GetWithdrawals(userID int64) ([]entity.Withdrawal, error)
}

type OrderRepository interface {
	CreateOrder(userID int64, orderNumber string) (int64, error)
	GetOrderByNumber(orderNumber string) (*entity.Order, error)
	GetOrdersByUserID(userID int64) ([]entity.Order, error)
	UpdateOrderStatus(orderNumber string, status string, accrual float64) error
}

type AccrualClient interface {
	GetAccrual(ctx context.Context, orderNumber string) (*dto.AccrualResponse, error)
}

const jwtSecret = "go-musthave-diploma-secret"

type AuthService struct {
	userRepo      UserRepository
	orderRepo     OrderRepository
	accrualClient AccrualClient
	accrualAddr   string
}

func NewAuthService(userRepo UserRepository, orderRepo OrderRepository, accrualClient AccrualClient, accrualAddr string) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		orderRepo:     orderRepo,
		accrualClient: accrualClient,
		accrualAddr:   accrualAddr,
	}
}

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func generateToken(userID int64, login string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"login":   login,
		"exp":     time.Now().Add(time.Hour * 24 * 30).Unix(),
	})
	return token.SignedString([]byte(jwtSecret))
}

func parseToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
}

func (s *AuthService) Register(login, password string) (string, error) {
	if login == "" || password == "" {
		return "", errors.New("требуются логин и пароль")
	}
	hashedPassword := hashPassword(password)
	_, err := s.userRepo.CreateUser(login, hashedPassword)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "duplicate key") || strings.Contains(errStr, "unique constraint") {
			return "", myErrors.ErrAlreadyExists
		}
		return "", err
	}
	user, err := s.userRepo.GetUserByLogin(login)
	if err != nil {
		return "", err
	}
	return generateToken(user.ID, user.Login)
}

func (s *AuthService) Login(login, password string) (string, error) {
	user, err := s.userRepo.GetUserByLogin(login)
	if err != nil {
		if errors.Is(err, myErrors.ErrNotFound) {
			return "", myErrors.ErrInvalidCreds
		}
		return "", err
	}
	hashedPassword := hashPassword(password)
	if user.Password != hashedPassword {
		return "", myErrors.ErrInvalidCreds
	}
	return generateToken(user.ID, user.Login)
}

func (s *AuthService) GetUserIDFromToken(tokenString string) (int64, error) {
	token, err := parseToken(tokenString)
	if err != nil || !token.Valid {
		return 0, myErrors.ErrUnauthorized
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, myErrors.ErrUnauthorized
	}
	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, myErrors.ErrUnauthorized
	}
	return int64(userIDFloat), nil
}

func (s *AuthService) UploadOrder(userID int64, orderNumber string) (int, error) {
	if !isValidOrderNumber(orderNumber) {
		return 0, myErrors.ErrInvalidOrder
	}
	existingOrder, err := s.orderRepo.GetOrderByNumber(orderNumber)
	if err != nil {
		if !errors.Is(err, myErrors.ErrNotFound) {
			return 0, err
		}
	}
	if existingOrder != nil {
		if existingOrder.UserID == userID {
			return http.StatusOK, nil
		}
		return 0, myErrors.ErrAlreadyExists
	}
	_, err = s.orderRepo.CreateOrder(userID, orderNumber)
	if err != nil {
		return 0, err
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.processAccrual(ctx, orderNumber)
	}()
	return http.StatusAccepted, nil
}

func isValidOrderNumber(orderNumber string) bool {
	if len(orderNumber) < 1 {
		return false
	}
	for _, c := range orderNumber {
		if c < '0' || c > '9' {
			return false
		}
	}
	return luhnCheck(orderNumber)
}

func luhnCheck(orderNumber string) bool {
	sum := 0
	isSecond := false
	for i := len(orderNumber) - 1; i >= 0; i-- {
		d := int(orderNumber[i] - '0')
		if isSecond {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		isSecond = !isSecond
	}
	return sum%10 == 0
}

func normalizeAccrualStatusForUser(status string) string {
	if status == "REGISTERED" {
		return string(entity.OrderStatusProcessing)
	}
	return status
}

func (s *AuthService) processAccrual(ctx context.Context, orderNumber string) {
	s.orderRepo.UpdateOrderStatus(orderNumber, string(entity.OrderStatusProcessing), 0)
	resp, err := s.accrualClient.GetAccrual(ctx, orderNumber)
	if err != nil {
		if errors.Is(err, myErrors.ErrNotFound) {
			s.orderRepo.UpdateOrderStatus(orderNumber, string(entity.OrderStatusNew), 0)
			return
		}
		s.orderRepo.UpdateOrderStatus(orderNumber, string(entity.OrderStatusInvalid), 0)
		return
	}
	st := normalizeAccrualStatusForUser(resp.Status)
	s.orderRepo.UpdateOrderStatus(orderNumber, st, resp.Accrual)
	if st == string(entity.OrderStatusProcessed) && resp.Accrual > 0 {
		order, err := s.orderRepo.GetOrderByNumber(orderNumber)
		if err == nil {
			s.userRepo.UpdateUserBalance(order.UserID, resp.Accrual)
		}
	}
}

func (s *AuthService) GetOrders(userID int64) ([]dto.Order, error) {
	orders, err := s.orderRepo.GetOrdersByUserID(userID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.Order, 0, len(orders))
	for _, o := range orders {
		result = append(result, dto.Order{
			Number:     o.Number,
			Status:     normalizeAccrualStatusForUser(o.Status),
			Accrual:    o.Accrual,
			UploadedAt: o.UploadedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

func (s *AuthService) GetBalance(userID int64) (*dto.BalanceResponse, error) {
	balance, spent, err := s.userRepo.GetUserBalance(userID)
	if err != nil {
		return nil, err
	}
	return &dto.BalanceResponse{
		Current:   balance,
		Withdrawn: spent,
	}, nil
}

func (s *AuthService) Withdraw(userID int64, orderNumber string, sum float64) error {
	if !isValidOrderNumber(orderNumber) {
		return myErrors.ErrInvalidOrder
	}
	if sum <= 0 {
		return myErrors.ErrInvalidBody
	}
	return s.userRepo.ProcessWithdrawal(userID, orderNumber, sum)
}

func (s *AuthService) GetWithdrawals(userID int64) ([]dto.Withdrawal, error) {
	withdrawals, err := s.userRepo.GetWithdrawals(userID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.Withdrawal, 0, len(withdrawals))
	for _, w := range withdrawals {
		result = append(result, dto.Withdrawal{
			Order:       w.OrderNumber,
			Sum:         w.Sum,
			ProcessedAt: w.ProcessedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

type AccrualService struct {
	client *http.Client
	addr   string
}

func NewAccrualService(addr string) *AccrualService {
	return &AccrualService{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		addr: addr,
	}
}

func (s *AccrualService) GetAccrual(ctx context.Context, orderNumber string) (*dto.AccrualResponse, error) {
	url := s.addr + "/api/orders/" + orderNumber
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, myErrors.ErrNotFound
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, errors.New("превышен лимит запросов")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, myErrors.ErrUnexpectedStatus
	}

	var result dto.AccrualResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
