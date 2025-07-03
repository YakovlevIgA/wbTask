package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/yakovleviga/brokerService/internal/service"
)

type Routers struct {
	Service service.OrderService
}

func NewRouters(r *Routers) *fiber.App {
	app := fiber.New()

	// Настройка CORS (разрешенные методы, заголовки, авторизация)
	app.Use(cors.New(cors.Config{
		AllowMethods:     "GET, POST, PUT, DELETE",
		AllowHeaders:     "Accept, Content-Type, X-REQUEST-SomeID",
		ExposeHeaders:    "Link",
		AllowCredentials: false,
		MaxAge:           300,
	}))

	app.Static("/", "./web")

	apiGroup := app.Group("/v1")

	apiGroup.Get("/orders/:order_uid", r.Service.GetOrder)

	return app
}
