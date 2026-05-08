package entity

import "time"

type User struct {
	ID       int64
	Login    string
	Password string
	Balance  float64
	Spent    float64
}

type Order struct {
	ID         int64
	UserID     int64
	Number     string
	Status     string
	Accrual    float64
	UploadedAt time.Time
}

type Withdrawal struct {
	ID          int64
	UserID      int64
	OrderNumber string
	Sum         float64
	ProcessedAt time.Time
}

type OrderStatus string

const (
	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusInvalid    OrderStatus = "INVALID"
	OrderStatusProcessed  OrderStatus = "PROCESSED"
)
