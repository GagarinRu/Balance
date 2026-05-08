package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"go-musthave-diploma-tpl/internal/dto"
	"go-musthave-diploma-tpl/internal/entity"
	"go-musthave-diploma-tpl/internal/service/mocks"
	myErrors "go-musthave-diploma-tpl/pkg/errors"
	"go.uber.org/mock/gomock"
)

func TestRegister(t *testing.T) {
	cases := []struct {
		name string

		login    string
		password string

		createUserID    int64
		createUserError error

		getUserByLoginResponse *entity.User
		getUserByLoginError    error

		expectedToken string
		expectedError error
	}{
		{
			name:     "normal",
			login:    "testuser",
			password: "testpass123",

			createUserID:    1,
			createUserError: nil,

			getUserByLoginResponse: &entity.User{ID: 1, Login: "testuser"},
			getUserByLoginError:    nil,

			expectedToken: "",
			expectedError: nil,
		},
		{
			name:     "duplicate login",
			login:    "existing",
			password: "pass123",

			createUserID:    0,
			createUserError: fmt.Errorf("pq: duplicate key"),

			getUserByLoginResponse: nil,
			getUserByLoginError:    nil,

			expectedToken: "",
			expectedError: myErrors.ErrAlreadyExists,
		},
		{
			name:     "empty login",
			login:    "",
			password: "pass123",

			createUserID:    0,
			createUserError: nil,

			getUserByLoginResponse: nil,
			getUserByLoginError:    nil,

			expectedToken: "",
			expectedError: errors.New("требуются логин и пароль"),
		},
		{
			name:     "empty password",
			login:    "testuser",
			password: "",

			createUserID:    0,
			createUserError: nil,

			getUserByLoginResponse: nil,
			getUserByLoginError:    nil,

			expectedToken: "",
			expectedError: errors.New("требуются логин и пароль"),
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("test case #%d: %s", i, tc.name), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			userRepoMock := mocks.NewMockUserRepository(ctrl)
			orderRepoMock := mocks.NewMockOrderRepository(ctrl)
			accrualClientMock := mocks.NewMockAccrualClient(ctrl)

			if tc.expectedError == nil || (tc.expectedError != nil && tc.expectedError.Error() != "требуются логин и пароль") {
				userRepoMock.EXPECT().
					CreateUser(tc.login, gomock.Any()).
					Times(1).
					Return(tc.createUserID, tc.createUserError)

				if tc.createUserError == nil {
					userRepoMock.EXPECT().
						GetUserByLogin(tc.login).
						Times(1).
						Return(tc.getUserByLoginResponse, tc.getUserByLoginError)
				}
			}

			as := NewAuthService(userRepoMock, orderRepoMock, accrualClientMock, "http://localhost:8081")

			token, err := as.Register(tc.login, tc.password)

			if tc.expectedError == nil {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				if token == "" {
					t.Errorf("expected non-empty token")
					return
				}
			} else {
				if err == nil || err.Error() != tc.expectedError.Error() {
					t.Errorf("expected error %v, got: %v", tc.expectedError, err)
					return
				}
			}
		})
	}
}

func TestLogin(t *testing.T) {
	cases := []struct {
		name string

		login    string
		password string

		getUserByLoginResponse *entity.User
		getUserByLoginError    error

		expectedToken string
		expectedError error
	}{
		{
			name:     "user not found",
			login:    "nonexistent",
			password: "pass123",

			getUserByLoginResponse: nil,
			getUserByLoginError:    myErrors.ErrNotFound,

			expectedToken: "",
			expectedError: myErrors.ErrInvalidCreds,
		},
		{
			name:     "invalid password",
			login:    "testuser",
			password: "wrongpass",

			getUserByLoginResponse: &entity.User{ID: 1, Login: "testuser", Password: "hashed_correct_password"},
			getUserByLoginError:    nil,

			expectedToken: "",
			expectedError: myErrors.ErrInvalidCreds,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("test case #%d: %s", i, tc.name), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			userRepoMock := mocks.NewMockUserRepository(ctrl)
			userRepoMock.EXPECT().
				GetUserByLogin(tc.login).
				Times(1).
				Return(tc.getUserByLoginResponse, tc.getUserByLoginError)

			orderRepoMock := mocks.NewMockOrderRepository(ctrl)
			accrualClientMock := mocks.NewMockAccrualClient(ctrl)

			as := NewAuthService(userRepoMock, orderRepoMock, accrualClientMock, "http://localhost:8081")

			token, err := as.Login(tc.login, tc.password)

			if tc.expectedError == nil {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				if token == "" {
					t.Errorf("expected non-empty token")
					return
				}
			} else {
				if err == nil || err.Error() != tc.expectedError.Error() {
					t.Errorf("expected error %v, got: %v", tc.expectedError, err)
					return
				}
			}
		})
	}
}

func TestUploadOrder(t *testing.T) {
	cases := []struct {
		name string

		userID      int64
		orderNumber string

		getOrderByNumberResponse *entity.Order
		getOrderByNumberError    error

		createOrderID    int64
		createOrderError error

		updateOrderStatusError error

		expectedStatusCode int
		expectedError      error
	}{
		{
			name:               "invalid order number - too short",
			userID:             1,
			orderNumber:        "123",
			expectedStatusCode: 0,
			expectedError:      myErrors.ErrInvalidOrder,
		},
		{
			name:        "order already exists for different user",
			userID:      1,
			orderNumber: "9278923470",

			getOrderByNumberResponse: &entity.Order{ID: 1, UserID: 2, Number: "9278923470"},
			getOrderByNumberError:    nil,

			expectedStatusCode: 0,
			expectedError:      myErrors.ErrAlreadyExists,
		},
		{
			name:        "new order - valid number",
			userID:      1,
			orderNumber: "9278923470",

			getOrderByNumberResponse: nil,
			getOrderByNumberError:    myErrors.ErrNotFound,

			createOrderID:    1,
			createOrderError: nil,

			updateOrderStatusError: nil,

			expectedStatusCode: 202,
			expectedError:      nil,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("test case #%d: %s", i, tc.name), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			userRepoMock := mocks.NewMockUserRepository(ctrl)
			orderRepoMock := mocks.NewMockOrderRepository(ctrl)
			accrualClientMock := mocks.NewMockAccrualClient(ctrl)

			if tc.expectedError != myErrors.ErrInvalidOrder {
				orderRepoMock.EXPECT().
					GetOrderByNumber(tc.orderNumber).
					AnyTimes().
					Return(tc.getOrderByNumberResponse, tc.getOrderByNumberError)

				orderRepoMock.EXPECT().
					CreateOrder(tc.userID, tc.orderNumber).
					AnyTimes().
					Return(tc.createOrderID, tc.createOrderError)

				orderRepoMock.EXPECT().
					UpdateOrderStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					AnyTimes().
					Return(tc.updateOrderStatusError)

				accrualClientMock.EXPECT().
					GetAccrual(gomock.Any(), tc.orderNumber).
					AnyTimes().
					Return(&dto.AccrualResponse{Status: "PROCESSED", Accrual: 500}, nil)
			}

			as := NewAuthService(userRepoMock, orderRepoMock, accrualClientMock, "http://localhost:8081")

			statusCode, err := as.UploadOrder(tc.userID, tc.orderNumber)

			if tc.expectedError != nil {
				if !errors.Is(err, tc.expectedError) {
					t.Errorf("expected error %v, got: %v", tc.expectedError, err)
					return
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				if statusCode != tc.expectedStatusCode {
					t.Errorf("expected status code %d, got: %d", tc.expectedStatusCode, statusCode)
					return
				}
			}

			time.Sleep(50 * time.Millisecond)
		})
	}
}

func TestGetBalance(t *testing.T) {
	cases := []struct {
		name string

		userID int64

		getUserBalanceResponseBalance float64
		getUserBalanceResponseSpent   float64
		getUserBalanceError           error

		expectedBalance *dto.BalanceResponse
		expectedError   error
	}{
		{
			name:   "normal",
			userID: 1,

			getUserBalanceResponseBalance: 500.5,
			getUserBalanceResponseSpent:   42,
			getUserBalanceError:           nil,

			expectedBalance: &dto.BalanceResponse{
				Current:   500.5,
				Withdrawn: 42,
			},
			expectedError: nil,
		},
		{
			name:   "zero balance",
			userID: 1,

			getUserBalanceResponseBalance: 0,
			getUserBalanceResponseSpent:   0,
			getUserBalanceError:           nil,

			expectedBalance: &dto.BalanceResponse{
				Current:   0,
				Withdrawn: 0,
			},
			expectedError: nil,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("test case #%d: %s", i, tc.name), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			userRepoMock := mocks.NewMockUserRepository(ctrl)
			userRepoMock.EXPECT().
				GetUserBalance(tc.userID).
				Times(1).
				Return(tc.getUserBalanceResponseBalance, tc.getUserBalanceResponseSpent, tc.getUserBalanceError)

			orderRepoMock := mocks.NewMockOrderRepository(ctrl)
			accrualClientMock := mocks.NewMockAccrualClient(ctrl)

			as := NewAuthService(userRepoMock, orderRepoMock, accrualClientMock, "http://localhost:8081")

			balance, err := as.GetBalance(tc.userID)

			if tc.expectedError == nil {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				if balance.Current != tc.expectedBalance.Current || balance.Withdrawn != tc.expectedBalance.Withdrawn {
					t.Errorf("expected balance %v, got: %v", tc.expectedBalance, balance)
					return
				}
			}
		})
	}
}

func TestWithdraw(t *testing.T) {
	cases := []struct {
		name string

		userID      int64
		orderNumber string
		sum         float64

		processWithdrawalError error

		expectedError error
	}{
		{
			name:        "invalid order number",
			userID:      1,
			orderNumber: "123",
			sum:         100,

			processWithdrawalError: nil,

			expectedError: myErrors.ErrInvalidOrder,
		},
		{
			name:        "zero sum",
			userID:      1,
			orderNumber: "9278923470",
			sum:         0,

			processWithdrawalError: nil,

			expectedError: myErrors.ErrInvalidBody,
		},
		{
			name:        "negative sum",
			userID:      1,
			orderNumber: "9278923470",
			sum:         -50,

			processWithdrawalError: nil,

			expectedError: myErrors.ErrInvalidBody,
		},
		{
			name:        "success",
			userID:      1,
			orderNumber: "9278923470",
			sum:         100,

			processWithdrawalError: nil,

			expectedError: nil,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("test case #%d: %s", i, tc.name), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			userRepoMock := mocks.NewMockUserRepository(ctrl)
			if tc.expectedError == nil {
				userRepoMock.EXPECT().
					ProcessWithdrawal(tc.userID, tc.orderNumber, tc.sum).
					Times(1).
					Return(tc.processWithdrawalError)
			}

			orderRepoMock := mocks.NewMockOrderRepository(ctrl)
			accrualClientMock := mocks.NewMockAccrualClient(ctrl)

			as := NewAuthService(userRepoMock, orderRepoMock, accrualClientMock, "http://localhost:8081")

			err := as.Withdraw(tc.userID, tc.orderNumber, tc.sum)

			if tc.expectedError != nil {
				if !errors.Is(err, tc.expectedError) {
					t.Errorf("expected error %v, got: %v", tc.expectedError, err)
					return
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
			}
		})
	}
}
