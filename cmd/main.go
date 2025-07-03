package main

import (
	"context"
	"fmt"
	"github.com/yakovleviga/brokerService/internal/cache"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"

	"github.com/yakovleviga/brokerService/internal/api"
	"github.com/yakovleviga/brokerService/internal/config"
	"github.com/yakovleviga/brokerService/internal/consumer"
	"github.com/yakovleviga/brokerService/internal/db"
	"github.com/yakovleviga/brokerService/internal/service"
)

func main() {
	// Загружаем .env, если есть
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			log.Println("Ошибка загрузки .env:", err)
		} else {
			log.Println("Загружен файл .env")
		}
	} else {
		log.Println("Файл .env не найден — берем окружение из environment compose-yml")
	}

	var cfg config.AppConfig
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal(errors.Wrap(err, "failed to load configuration"))
	}

	c := cache.NewCache()

	ctx := context.Background()

	if err := db.RunMigrations(cfg.PostgreSQL); err != nil {
		log.Fatal(err)
	}

	repository, err := db.NewRepository(ctx, cfg.PostgreSQL)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to initialize repository"))
	}

	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := repository.Ping(ctxPing); err != nil {
		log.Fatal(errors.Wrap(err, "failed to ping database"))
	}
	log.Println("Подключение к базе данных успешно")

	serviceInstance := service.NewService(repository, c)

	app := api.NewRouters(&api.Routers{Service: *serviceInstance})

	LoadCacheFromDB(repository, c)
	cache.PrintCache(c)

	go func() {
		fmt.Printf("Starting server on %s\n", cfg.Rest.ListenAddress)
		if err := app.Listen(cfg.Rest.ListenAddress); err != nil {
			log.Fatal(errors.Wrap(err, "failed to start server"))
		}
	}()

	go consumer.ConsumeKafka(repository, c)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	log.Println("Shutting down gracefully...")
	// TODO: Добавить логику graceful shutdown, если нужно
}

func LoadCacheFromDB(repo db.Repository, c *cache.Cache) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	orders, err := repo.GetAllOrders(ctx)
	if err != nil {
		log.Fatalf("failed to load orders from DB: %v", err)
	}

	for _, order := range orders {
		c.Set(order)
	}

	log.Printf("Cache loaded with %d orders from DB", len(orders))
}
