package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-musthave-diploma-tpl/internal/config"
	"go-musthave-diploma-tpl/internal/handler"
	"go-musthave-diploma-tpl/internal/logger"
	"go-musthave-diploma-tpl/internal/repository/postgres"
	"go-musthave-diploma-tpl/internal/router"
	"go-musthave-diploma-tpl/internal/service"

	"go.uber.org/zap"
)

func main() {
	var (
		addr        string
		databaseURI string
		accrualAddr string
	)
	flag.StringVar(&addr, "a", ":8080", "Server address")
	flag.StringVar(&databaseURI, "d", "", "Database URI")
	flag.StringVar(&accrualAddr, "r", "http://localhost:8081", "Accrual system address")
	flag.Parse()

	cfg := config.NewConfig()
	if addr != ":8080" {
		cfg.RunAddress = addr
	}
	if databaseURI != "" {
		cfg.DatabaseURI = databaseURI
	}
	if accrualAddr != "http://localhost:8081" {
		cfg.AccrualSystemAddr = accrualAddr
	}

	if err := logger.Initialize("info"); err != nil {
		logger.Log.Fatal("Не удалось инициализировать logger", zap.Error(err))
	}
	defer logger.Log.Sync()

	if cfg.DatabaseURI == "" {
		logger.Log.Fatal("Требуется URI базы данных")
	}

	repo, err := postgres.NewStorage(cfg.DatabaseURI)
	if err != nil {
		logger.Log.Fatal("Не удалось подключиться к базе данных", zap.Error(err))
	}
	defer repo.Close()

	accrualClient := service.NewAccrualService(cfg.AccrualSystemAddr)
	authService := service.NewAuthService(repo, repo, accrualClient, cfg.AccrualSystemAddr)
	h := handler.NewHandler(authService)
	r := router.Init(h)

	server := &http.Server{
		Addr:         cfg.RunAddress,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		logger.Log.Info("Запуск сервера", zap.String("address", cfg.RunAddress))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Ошибка сервера", zap.Error(err))
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Log.Info("Выключение сервера")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Log.Fatal("Не удалось выключить сервер", zap.Error(err))
	}
	logger.Log.Info("Сервер успешно остановлен")
}
