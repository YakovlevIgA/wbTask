package db

import (
	"context"
	"database/sql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"

	"github.com/yakovleviga/brokerService/internal/models"
	"log"

	"fmt"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/yakovleviga/brokerService/internal/config"
)

const (
	orderQuery = `
        SELECT order_uid, track_number, entry, locale, internal_signature,
               customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        FROM orders
        WHERE order_uid = $1
    `
	itemsQuery = `
        SELECT chrt_id, track_number, price, rid, name, sale, size, total_price,
               nm_id, brand, status
        FROM items
        WHERE order_uid = $1
    `
	paymentQuery = `
        SELECT transaction, request_id, currency, provider, amount,
               payment_dt, bank, delivery_cost, goods_total, custom_fee
        FROM payment
        WHERE order_uid = $1
    `
	deliveryQuery = `
        SELECT name, phone, zip, city, address, region, email
        FROM delivery
        WHERE order_uid = $1
    `
)

type repository struct {
	pool *pgxpool.Pool
}

type Repository interface {
	Ping(ctx context.Context) error
	GetFullOrder(ctx context.Context, orderUID string) (*FullOrder, error)
	InsertOrder(ctx context.Context, order models.Order) (err error)
	GetAllOrders(ctx context.Context) ([]FullOrder, error)
}

func NewRepository(ctx context.Context, cfg config.PostgreSQL) (Repository, error) {
	// Формируем строку подключения
	connString := fmt.Sprintf(
		`user=%s password=%s host=%s port=%d dbname=%s sslmode=%s 
        pool_max_conns=%d pool_max_conn_lifetime=%s pool_max_conn_idle_time=%s`,
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
		cfg.PoolMaxConns,
		cfg.PoolMaxConnLifetime.String(),
		cfg.PoolMaxConnIdleTime.String(),
	)

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse PostgreSQL config")
	}

	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheDescribe

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create PostgreSQL connection pool")
	}

	return &repository{pool}, nil
}

func (r *repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func RunMigrations(cfg config.PostgreSQL) error {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return errors.Wrap(err, "failed to open database for migration")
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return errors.Wrap(err, "failed to create migration driver")
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://internal/db/migrations",
		"postgres", driver)
	if err != nil {
		return errors.Wrap(err, "failed to create migrate instance")
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return errors.Wrap(err, "migration up failed")
	}

	log.Println("✅ Migrations applied successfully")
	return nil
}

func (r *repository) GetFullOrder(ctx context.Context, orderUID string) (*FullOrder, error) {
	var order FullOrder

	err := r.pool.QueryRow(ctx, orderQuery, orderUID).Scan(
		&order.OrderUID,
		&order.TrackNumber,
		&order.Entry,
		&order.Locale,
		&order.InternalSignature,
		&order.CustomerID,
		&order.DeliveryService,
		&order.ShardKey,
		&order.SmID,
		&order.DateCreated,
		&order.OofShard,
	)
	if err != nil {
		return nil, err
	}

	err = r.pool.QueryRow(ctx, deliveryQuery, orderUID).Scan(
		&order.Delivery.Name,
		&order.Delivery.Phone,
		&order.Delivery.Zip,
		&order.Delivery.City,
		&order.Delivery.Address,
		&order.Delivery.Region,
		&order.Delivery.Email,
	)
	if err != nil {
		return nil, err
	}

	err = r.pool.QueryRow(ctx, paymentQuery, orderUID).Scan(
		&order.Payment.Transaction,
		&order.Payment.RequestID,
		&order.Payment.Currency,
		&order.Payment.Provider,
		&order.Payment.Amount,
		&order.Payment.PaymentDT,
		&order.Payment.Bank,
		&order.Payment.DeliveryCost,
		&order.Payment.GoodsTotal,
		&order.Payment.CustomFee,
	)
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, itemsQuery, orderUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item Item
		if err := rows.Scan(
			&item.ChrtID,
			&item.TrackNumber,
			&item.Price,
			&item.Rid,
			&item.Name,
			&item.Sale,
			&item.Size,
			&item.TotalPrice,
			&item.NmID,
			&item.Brand,
			&item.Status,
		); err != nil {
			return nil, err
		}
		order.Items = append(order.Items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *repository) InsertOrder(ctx context.Context, order models.Order) (err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("start transaction: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	_, err = tx.Exec(ctx, `
        INSERT INTO orders (
            order_uid, track_number, entry, locale, internal_signature, customer_id,
            delivery_service, shardkey, sm_id, date_created, oof_shard
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
    `,
		order.OrderUID,
		order.TrackNumber,
		order.Entry,
		order.Locale,
		order.InternalSignature,
		order.CustomerID,
		order.DeliveryService,
		order.ShardKey,
		order.SmID,
		order.DateCreated,
		order.OofShard,
	)
	if err != nil {
		return fmt.Errorf("insert orders: %w", err)
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO delivery (
            order_uid, name, phone, zip, city, address, region, email
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
    `,
		order.OrderUID,
		order.Delivery.Name,
		order.Delivery.Phone,
		order.Delivery.Zip,
		order.Delivery.City,
		order.Delivery.Address,
		order.Delivery.Region,
		order.Delivery.Email,
	)
	if err != nil {
		return fmt.Errorf("insert delivery: %w", err)
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO payment (
            order_uid, transaction, request_id, currency, provider,
            amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
    `,
		order.OrderUID,
		order.Payment.Transaction,
		order.Payment.RequestID,
		order.Payment.Currency,
		order.Payment.Provider,
		order.Payment.Amount,
		order.Payment.PaymentDT,
		order.Payment.Bank,
		order.Payment.DeliveryCost,
		order.Payment.GoodsTotal,
		order.Payment.CustomFee,
	)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}

	for _, item := range order.Items {
		_, err = tx.Exec(ctx, `
            INSERT INTO items (
                order_uid, chrt_id, track_number, price, rid,
                name, sale, size, total_price, nm_id, brand, status
            ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
        `,
			order.OrderUID,
			item.ChrtID,
			item.TrackNumber,
			item.Price,
			item.Rid,
			item.Name,
			item.Sale,
			item.Size,
			item.TotalPrice,
			item.NmID,
			item.Brand,
			item.Status,
		)
		if err != nil {
			return fmt.Errorf("insert item: %w", err)
		}
	}

	return nil
}

func (r *repository) GetAllOrders(ctx context.Context) ([]FullOrder, error) {

	rows, err := r.pool.Query(ctx, `
SELECT o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature, o.customer_id,
       o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
       d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
       pay.transaction, pay.request_id, pay.currency, pay.provider, pay.amount,
       pay.payment_dt, pay.bank, pay.delivery_cost, pay.goods_total, pay.custom_fee
FROM orders o
JOIN delivery d ON o.order_uid = d.order_uid
JOIN payment pay ON o.order_uid = pay.order_uid
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []FullOrder

	for rows.Next() {
		var o FullOrder
		var d Delivery
		var pmt Payment

		err := rows.Scan(
			&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature, &o.CustomerID,
			&o.DeliveryService, &o.ShardKey, &o.SmID, &o.DateCreated, &o.OofShard,
			&d.Name, &d.Phone, &d.Zip, &d.City, &d.Address, &d.Region, &d.Email,
			&pmt.Transaction, &pmt.RequestID, &pmt.Currency, &pmt.Provider, &pmt.Amount,
			&pmt.PaymentDT, &pmt.Bank, &pmt.DeliveryCost, &pmt.GoodsTotal, &pmt.CustomFee,
		)
		if err != nil {
			return nil, err
		}
		o.Delivery = d
		o.Payment = pmt

		itemRows, err := r.pool.Query(ctx, `
SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
FROM items
WHERE order_uid = $1
`, o.OrderUID)
		if err != nil {
			return nil, err
		}

		var items []Item
		for itemRows.Next() {
			var it Item
			err := itemRows.Scan(
				&it.ChrtID, &it.TrackNumber, &it.Price, &it.Rid, &it.Name,
				&it.Sale, &it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status,
			)
			if err != nil {
				itemRows.Close()
				return nil, err
			}
			items = append(items, it)
		}
		itemRows.Close()

		o.Items = items

		orders = append(orders, o)
	}

	return orders, nil
}
