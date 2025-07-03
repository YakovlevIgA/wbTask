package service

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yakovleviga/brokerService/internal/cache"
	"github.com/yakovleviga/brokerService/internal/db"
)

type OrderService struct {
	db    db.Repository
	cache *cache.Cache
}

func NewService(repository db.Repository, cache *cache.Cache) *OrderService {
	return &OrderService{
		db:    repository,
		cache: cache,
	}
}

func (s *OrderService) GetOrder(c *fiber.Ctx) error {
	orderUID := c.Params("order_uid")
	if orderUID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing order_uid"})
	}

	order, found := s.cache.Get(orderUID)
	if !found {

		orderPtr, err := s.db.GetFullOrder(c.Context(), orderUID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
		}
		order = *orderPtr
		s.cache.Set(order)
	}

	return c.JSON(order)
}
