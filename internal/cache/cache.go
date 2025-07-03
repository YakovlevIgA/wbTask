package cache

import (
	"fmt"
	"sync"

	"github.com/yakovleviga/brokerService/internal/db"
)

type Cache struct {
	mu     sync.RWMutex
	orders map[string]db.FullOrder
}

func NewCache() *Cache {
	return &Cache{
		orders: make(map[string]db.FullOrder),
	}
}

func (c *Cache) Set(order db.FullOrder) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.orders[order.OrderUID] = order
}

func (c *Cache) Get(orderUID string) (db.FullOrder, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	order, exists := c.orders[orderUID]
	return order, exists
}

func (c *Cache) Delete(orderUID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.orders, orderUID)
}

func (c *Cache) GetAll() []db.FullOrder {
	c.mu.RLock()
	defer c.mu.RUnlock()
	all := make([]db.FullOrder, 0, len(c.orders))
	for _, order := range c.orders {
		all = append(all, order)
	}
	return all
}

func PrintCache(c *Cache) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for k, v := range c.orders {
		fmt.Printf("OrderUID: %s, Order: %+v\n", k, v)
	}
}
