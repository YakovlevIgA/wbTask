package main

import (
	"context"
	"github.com/segmentio/kafka-go"
	"log"
	"time"
)

func main() {
	
	w := &kafka.Writer{
		Addr:  kafka.TCP("kafka:9092"),
		Topic: "orders",
	}
	defer w.Close()

	ctx := context.Background()

	// Подготовленные 4 набора данных (строки JSON)
	messages := []kafka.Message{
		{
			Partition: 0,
			Key:       []byte("order001"),
			Value: []byte(`{
				"order_uid": "order001",
				"track_number": "TRACK001",
				"entry": "WBIL",
				"delivery": {
					"name": "Ivan Petrov",
					"phone": "+79000000001",
					"zip": "123456",
					"city": "Moscow",
					"address": "Lenina 1",
					"region": "Moscow",
					"email": "ivan@example.com"
				},
				"payment": {
					"transaction": "order001",
					"request_id": "",
					"currency": "RUB",
					"provider": "wbpay",
					"amount": 1000,
					"payment_dt": 1637900001,
					"bank": "sber",
					"delivery_cost": 200,
					"goods_total": 800,
					"custom_fee": 0
				},
				"items": [{
					"chrt_id": 1111111,
					"track_number": "TRACK001",
					"price": 800,
					"rid": "rid001",
					"name": "T-shirt",
					"sale": 0,
					"size": "M",
					"total_price": 800,
					"nm_id": 1010101,
					"brand": "Uniqlo",
					"status": 202
				}],
				"locale": "ru",
				"internal_signature": "",
				"customer_id": "cust001",
				"delivery_service": "cdek",
				"shardkey": "1",
				"sm_id": 1,
				"date_created": "2021-11-26T06:22:19Z",
				"oof_shard": "1"
			}`),
		},
		{
			Partition: 0,
			Key:       []byte("order002"),
			Value: []byte(`{
				"order_uid": "order002",
				"track_number": "TRACK002",
				"entry": "WBIL",
				"delivery": {
					"name": "Anna Ivanova",
					"phone": "+79000000002",
					"zip": "654321",
					"city": "Saint Petersburg",
					"address": "Nevsky 100",
					"region": "Leningrad",
					"email": "anna@example.com"
				},
				"payment": {
					"transaction": "order002",
					"request_id": "",
					"currency": "RUB",
					"provider": "wbpay",
					"amount": 2500,
					"payment_dt": 1637900002,
					"bank": "tinkoff",
					"delivery_cost": 300,
					"goods_total": 2200,
					"custom_fee": 0
				},
				"items": [{
					"chrt_id": 2222222,
					"track_number": "TRACK002",
					"price": 2200,
					"rid": "rid002",
					"name": "Shoes",
					"sale": 10,
					"size": "42",
					"total_price": 1980,
					"nm_id": 2020202,
					"brand": "Nike",
					"status": 202
				}],
				"locale": "ru",
				"internal_signature": "",
				"customer_id": "cust002",
				"delivery_service": "dhl",
				"shardkey": "2",
				"sm_id": 2,
				"date_created": "2021-11-27T06:22:19Z",
				"oof_shard": "2"
			}`),
		},
		{
			Partition: 0,
			Key:       []byte("order003"),
			Value: []byte(`{
				"order_uid": "order003",
				"track_number": "TRACK003",
				"entry": "WBIL",
				"delivery": {
					"name": "Sergey Smirnov",
					"phone": "+79000000003",
					"zip": "789123",
					"city": "Kazan",
					"address": "Pushkina 5",
					"region": "Tatarstan",
					"email": "sergey@example.com"
				},
				"payment": {
					"transaction": "order003",
					"request_id": "",
					"currency": "RUB",
					"provider": "wbpay",
					"amount": 3500,
					"payment_dt": 1637900003,
					"bank": "vtb",
					"delivery_cost": 500,
					"goods_total": 3000,
					"custom_fee": 0
				},
				"items": [{
					"chrt_id": 3333333,
					"track_number": "TRACK003",
					"price": 3000,
					"rid": "rid003",
					"name": "Laptop",
					"sale": 0,
					"size": "-",
					"total_price": 3000,
					"nm_id": 3030303,
					"brand": "Lenovo",
					"status": 202
				}],
				"locale": "ru",
				"internal_signature": "",
				"customer_id": "cust003",
				"delivery_service": "boxberry",
				"shardkey": "3",
				"sm_id": 3,
				"date_created": "2021-11-28T06:22:19Z",
				"oof_shard": "3"
			}`),
		},
		{
			Partition: 0,
			Key:       []byte("order004_invalid"),
			Value: []byte(`{
				"track_number": "TRACK004",
				"entry": "WBIL",
				"delivery": {
					"name": "Elena Sidorova",
					"phone": "+79000000004",
					"zip": "456789",
					"city": "Novosibirsk",
					"address": "Sovetskaya 10",
					"region": "Novosibirsk",
					"email": "elena@example.com"
				},
				"payment": {
					"transaction": "order004",
					"request_id": "",
					"currency": "RUB",
					"provider": "wbpay",
					"amount": 1500,
					"payment_dt": 1637900004,
					"bank": "gazprom",
					"delivery_cost": 200,
					"goods_total": 1300,
					"custom_fee": 0
				},
				"items": [{
					"chrt_id": 4444444,
					"track_number": "TRACK004",
					"price": 1300,
					"rid": "rid004",
					"name": "Backpack",
					"sale": 5,
					"size": "L",
					"total_price": 1235,
					"nm_id": 4040404,
					"brand": "Adidas",
					"status": 202
				}],
				"locale": "ru",
				"internal_signature": "",
				"customer_id": "cust004",
				"delivery_service": "cdek",
				"shardkey": "4",
				"sm_id": 4,
				"date_created": "2021-11-29T06:22:19Z",
				"oof_shard": "4"
			}`),
		},
	}

	// Отправка всех сообщений
	for _, msg := range messages {
		err := w.WriteMessages(ctx, msg)
		if err != nil {
			log.Printf("failed to write message %s: %v", msg.Key, err)
		} else {
			log.Printf("message %s written successfully", msg.Key)
		}
		time.Sleep(500 * time.Millisecond)
	}
}
