package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/yakovleviga/brokerService/internal/cache"
	"github.com/yakovleviga/brokerService/internal/db"
	"github.com/yakovleviga/brokerService/internal/models"
)

func FullOrderToModelOrder(fo db.FullOrder) models.Order {
	items := make([]models.Item, len(fo.Items))
	for i, item := range fo.Items {
		items[i] = models.Item{
			ChrtID:      int(item.ChrtID),
			TrackNumber: item.TrackNumber,
			Price:       item.Price,
			Rid:         item.Rid,
			Name:        item.Name,
			Sale:        item.Sale,
			Size:        item.Size,
			TotalPrice:  item.TotalPrice,
			NmID:        item.NmID,
			Brand:       item.Brand,
			Status:      item.Status,
		}
	}

	return models.Order{
		OrderUID:          fo.OrderUID,
		TrackNumber:       fo.TrackNumber,
		Entry:             fo.Entry,
		Locale:            fo.Locale,
		InternalSignature: fo.InternalSignature,
		CustomerID:        fo.CustomerID,
		DeliveryService:   fo.DeliveryService,
		ShardKey:          fo.ShardKey,
		SmID:              fo.SmID,
		DateCreated:       fo.DateCreated,
		OofShard:          fo.OofShard,
		Delivery: models.Delivery{
			Name:    fo.Delivery.Name,
			Phone:   fo.Delivery.Phone,
			Zip:     fo.Delivery.Zip,
			City:    fo.Delivery.City,
			Address: fo.Delivery.Address,
			Region:  fo.Delivery.Region,
			Email:   fo.Delivery.Email,
		},
		Payment: models.Payment{
			Transaction:  fo.Payment.Transaction,
			RequestID:    fo.Payment.RequestID,
			Currency:     fo.Payment.Currency,
			Provider:     fo.Payment.Provider,
			Amount:       fo.Payment.Amount,
			PaymentDT:    fo.Payment.PaymentDT,
			Bank:         fo.Payment.Bank,
			DeliveryCost: fo.Payment.DeliveryCost,
			GoodsTotal:   fo.Payment.GoodsTotal,
			CustomFee:    fo.Payment.CustomFee,
		},
		Items: items,
	}
}

func ConsumeKafka(repo db.Repository, cache *cache.Cache) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{"kafka:9092"},
		Topic:     "orders",
		Partition: 0,
		MinBytes:  1,
		MaxBytes:  10e6,
	})
	defer r.Close()

	fmt.Println("Consumer started, waiting for messages...")

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		m, err := r.ReadMessage(ctx)
		cancel()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			time.Sleep(time.Second)
			continue
		}

		var fullOrder db.FullOrder
		if err := json.Unmarshal(m.Value, &fullOrder); err != nil {
			log.Println("JSON parse error:", err)
			continue
		}

		if fullOrder.OrderUID == "" {
			log.Println("FullOrder missing order_uid, skipping")
			continue
		}

		orderModel := FullOrderToModelOrder(fullOrder)

		ctxDB, cancelDB := context.WithTimeout(context.Background(), 5*time.Second)
		err = repo.InsertOrder(ctxDB, orderModel)
		cancelDB()

		if err != nil {
			log.Println("DB insert error:", err)
			continue
		}

		cache.Set(fullOrder)

		if err := r.CommitMessages(context.Background(), m); err != nil {
			log.Println("Commit error:", err)
		} else {
			log.Printf("Order %s saved successfully", fullOrder.OrderUID)
		}

		fmt.Printf("Received message at offset %d: key=%s\n", m.Offset, string(m.Key))
	}
}
