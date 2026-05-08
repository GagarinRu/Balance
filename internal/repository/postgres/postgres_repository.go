package postgres

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"time"

	"go-musthave-diploma-tpl/internal/entity"
	myErrors "go-musthave-diploma-tpl/pkg/errors"

	_ "github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

func NewStorage(databaseURI string) (*Storage, error) {
	db, err := sql.Open("postgres", databaseURI)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	if err := applyMigrations(db); err != nil {
		return nil, err
	}
	return &Storage{db: db}, nil
}

func applyMigrations(db *sql.DB) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	migrationFile := os.Getenv("MIGRATION_FILE")
	if migrationFile == "" {
		migrationFile = "/migrations/001_init.sql"
	}
	data, err := os.ReadFile(migrationFile)
	if err != nil {
		data, err = os.ReadFile(wd + "/migrations/001_init.sql")
		if err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = db.ExecContext(ctx, string(data))
	return err
}

func (r *Storage) CreateUser(login, passwordHash string) (int64, error) {
	var id int64
	err := r.db.QueryRow(
		`INSERT INTO users (login, password, balance, spent) VALUES ($1, $2, 0, 0) RETURNING id`,
		login, passwordHash,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Storage) GetUserByLogin(login string) (*entity.User, error) {
	var u entity.User
	err := r.db.QueryRow(
		`SELECT id, login, password, balance, spent FROM users WHERE login = $1`,
		login,
	).Scan(&u.ID, &u.Login, &u.Password, &u.Balance, &u.Spent)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, myErrors.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *Storage) CreateOrder(userID int64, orderNumber string) (int64, error) {
	var id int64
	err := r.db.QueryRow(
		`INSERT INTO orders (user_id, number, status, accrual, uploaded_at) 
		 VALUES ($1, $2, 'NEW', 0, $3) RETURNING id`,
		userID, orderNumber, time.Now(),
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Storage) GetOrderByNumber(orderNumber string) (*entity.Order, error) {
	var o entity.Order
	var uploadedAt time.Time
	err := r.db.QueryRow(
		`SELECT id, user_id, number, status, accrual, uploaded_at FROM orders WHERE number = $1`,
		orderNumber,
	).Scan(&o.ID, &o.UserID, &o.Number, &o.Status, &o.Accrual, &uploadedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, myErrors.ErrNotFound
		}
		return nil, err
	}
	o.UploadedAt = uploadedAt
	return &o, nil
}

func (r *Storage) GetOrdersByUserID(userID int64) ([]entity.Order, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, number, status, accrual, uploaded_at FROM orders 
		 WHERE user_id = $1 ORDER BY uploaded_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []entity.Order
	for rows.Next() {
		var o entity.Order
		var uploadedAt time.Time
		if err := rows.Scan(&o.ID, &o.UserID, &o.Number, &o.Status, &o.Accrual, &uploadedAt); err != nil {
			return nil, err
		}
		o.UploadedAt = uploadedAt
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *Storage) UpdateOrderStatus(orderNumber string, status string, accrual float64) error {
	_, err := r.db.Exec(
		`UPDATE orders SET status = $1, accrual = $2 WHERE number = $3`,
		status, accrual, orderNumber,
	)
	return err
}

func (r *Storage) GetUserBalance(userID int64) (float64, float64, error) {
	var balance, spent float64
	err := r.db.QueryRow(
		`SELECT balance, spent FROM users WHERE id = $1`,
		userID,
	).Scan(&balance, &spent)
	if err != nil {
		return 0, 0, err
	}
	return balance, spent, nil
}

func (r *Storage) UpdateUserBalance(userID int64, amount float64) error {
	_, err := r.db.Exec(
		`UPDATE users SET balance = balance + $1 WHERE id = $2`,
		amount, userID,
	)
	return err
}

func (r *Storage) ProcessWithdrawal(userID int64, orderNumber string, sum float64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var balance float64
	err = tx.QueryRow(`SELECT balance FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&balance)
	if err != nil {
		return err
	}
	if balance < sum {
		return myErrors.ErrInsufficientFunds
	}

	_, err = tx.Exec(
		`UPDATE users SET balance = balance - $1, spent = spent + $1 WHERE id = $2`,
		sum, userID,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		`INSERT INTO withdrawals (user_id, order_number, sum, processed_at) VALUES ($1, $2, $3, $4)`,
		userID, orderNumber, sum, time.Now(),
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Storage) GetWithdrawals(userID int64) ([]entity.Withdrawal, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, order_number, sum, processed_at FROM withdrawals 
		 WHERE user_id = $1 ORDER BY processed_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdrawals []entity.Withdrawal
	for rows.Next() {
		var w entity.Withdrawal
		var processedAt time.Time
		if err := rows.Scan(&w.ID, &w.UserID, &w.OrderNumber, &w.Sum, &processedAt); err != nil {
			return nil, err
		}
		w.ProcessedAt = processedAt
		withdrawals = append(withdrawals, w)
	}
	return withdrawals, rows.Err()
}

func (r *Storage) Close() error {
	return r.db.Close()
}

func (r *Storage) Ping() error {
	return r.db.Ping()
}
