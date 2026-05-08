package errors

import "errors"

var (
	ErrNotFound          = errors.New("не найдено")
	ErrAlreadyExists     = errors.New("уже существует")
	ErrInvalidCreds      = errors.New("некорректные учётные данные")
	ErrUnauthorized      = errors.New("требуется авторизация")
	ErrInsufficientFunds = errors.New("недостаточно средств")
	ErrInvalidOrder      = errors.New("некорректный заказ")
	ErrInternalError     = errors.New("внутренняя ошибка")
	ErrInvalidBody       = errors.New("некорректное тело запроса")
	ErrUnexpectedStatus  = errors.New("неожиданный статус код")
)
