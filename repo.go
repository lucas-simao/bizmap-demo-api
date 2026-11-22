package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	insertCustomerSQL = `INSERT INTO customers (id, email, name, tier) VALUES ($1, $2, $3, 'standard')`
	selectCustomerSQL = `SELECT id, email, name, tier FROM customers WHERE id = $1`
	listProductsSQL   = `SELECT sku, name, price_cents, stock_on_hand - stock_reserved FROM products WHERE active ORDER BY name`
	selectPriceSQL    = `SELECT price_cents FROM products WHERE sku = $1`
	reserveStockSQL   = `UPDATE products SET stock_reserved = stock_reserved + $2 WHERE sku = $1 AND stock_on_hand - stock_reserved >= $2`
	releaseStockSQL   = `UPDATE products SET stock_reserved = stock_reserved - oi.qty FROM order_items oi WHERE oi.order_id = $1 AND products.sku = oi.sku`
	commitStockSQL    = `UPDATE products SET stock_on_hand = stock_on_hand - oi.qty, stock_reserved = stock_reserved - oi.qty FROM order_items oi WHERE oi.order_id = $1 AND products.sku = oi.sku`
	insertOrderSQL    = `INSERT INTO orders (id, customer_id, status, subtotal_cents, discount_cents, total_cents) VALUES ($1, $2, 'pending', $3, $4, $5)`
	insertItemSQL     = `INSERT INTO order_items (order_id, sku, qty, unit_price_cents) VALUES ($1, $2, $3, $4)`
	selectOrderSQL    = `SELECT id, customer_id, status, total_cents FROM orders WHERE id = $1`
	setStatusSQL      = `UPDATE orders SET status = $2, updated_at = now() WHERE id = $1`
	insertPaymentSQL  = `INSERT INTO payments (order_id, provider_ref, amount_cents) VALUES ($1, $2, $3)`
	insertShipmentSQL = `INSERT INTO shipments (order_id, carrier, tracking_code) VALUES ($1, $2, $3)`
	expirePendingSQL  = `UPDATE orders SET status = 'expired', updated_at = now() WHERE status = 'pending' AND created_at < now() - interval '30 minutes' RETURNING id`
)

type Customer struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Tier  string `json:"tier"`
}

type Product struct {
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	PriceCents int64  `json:"price_cents"`
	Available  int    `json:"available"`
}

type Order struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Status     string `json:"status"`
	TotalCents int64  `json:"total_cents"`
}

type Line struct {
	SKU string `json:"sku"`
	Qty int    `json:"qty"`
}

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

// InsertCustomer signs up a new customer on the standard tier.
func (r *Repo) InsertCustomer(ctx context.Context, email, name string) (string, error) {
	id := fmt.Sprintf("cus_%d", time.Now().UnixNano())
	_, err := r.db.Exec(ctx, insertCustomerSQL, id, email, name)
	return id, err
}

// GetCustomer loads one customer, including the loyalty tier used for discounts.
func (r *Repo) GetCustomer(ctx context.Context, id string) (Customer, error) {
	var c Customer
	err := r.db.QueryRow(ctx, selectCustomerSQL, id).Scan(&c.ID, &c.Email, &c.Name, &c.Tier)
	return c, err
}

// ListProducts returns the active catalog with the units still available.
func (r *Repo) ListProducts(ctx context.Context) ([]Product, error) {
	rows, err := r.db.Query(ctx, listProductsSQL)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Product])
}

// PlaceOrder reserves stock for every line and stores the priced order, in
// one transaction: if any SKU is short, nothing is reserved.
func (r *Repo) PlaceOrder(ctx context.Context, c Customer, lines []Line) (Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)

	var subtotal int64
	units := 0
	prices := make([]int64, len(lines))
	for i, l := range lines {
		if err := tx.QueryRow(ctx, selectPriceSQL, l.SKU).Scan(&prices[i]); err != nil {
			return Order{}, err
		}
		tag, err := tx.Exec(ctx, reserveStockSQL, l.SKU, l.Qty)
		if err != nil {
			return Order{}, err
		}
		if tag.RowsAffected() == 0 {
			return Order{}, fmt.Errorf("out of stock: %s", l.SKU)
		}
		subtotal += prices[i] * int64(l.Qty)
		units += l.Qty
	}
	discount := DiscountCents(subtotal, units, c.Tier)
	o := Order{ID: fmt.Sprintf("ord_%d", time.Now().UnixNano()), CustomerID: c.ID, Status: "pending", TotalCents: subtotal - discount}
	if _, err := tx.Exec(ctx, insertOrderSQL, o.ID, c.ID, subtotal, discount, o.TotalCents); err != nil {
		return Order{}, err
	}
	for i, l := range lines {
		if _, err := tx.Exec(ctx, insertItemSQL, o.ID, l.SKU, l.Qty, prices[i]); err != nil {
			return Order{}, err
		}
	}
	return o, tx.Commit(ctx)
}

// GetOrder loads one order by id.
func (r *Repo) GetOrder(ctx context.Context, id string) (Order, error) {
	var o Order
	err := r.db.QueryRow(ctx, selectOrderSQL, id).Scan(&o.ID, &o.CustomerID, &o.Status, &o.TotalCents)
	return o, err
}

// RecordPayment stores the provider charge, marks the order paid and turns
// the reservation into a stock decrement.
func (r *Repo) RecordPayment(ctx context.Context, orderID, providerRef string, amount int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, insertPaymentSQL, orderID, providerRef, amount); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, setStatusSQL, orderID, "paid"); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, commitStockSQL, orderID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RecordShipment stores the tracking code and marks the order shipped.
func (r *Repo) RecordShipment(ctx context.Context, orderID, carrier, tracking string) error {
	if _, err := r.db.Exec(ctx, insertShipmentSQL, orderID, carrier, tracking); err != nil {
		return err
	}
	_, err := r.db.Exec(ctx, setStatusSQL, orderID, "shipped")
	return err
}

// CancelOrder marks the order cancelled and gives its reserved units back.
func (r *Repo) CancelOrder(ctx context.Context, orderID string) error {
	if _, err := r.db.Exec(ctx, releaseStockSQL, orderID); err != nil {
		return err
	}
	_, err := r.db.Exec(ctx, setStatusSQL, orderID, "cancelled")
	return err
}

// ExpirePending expires unpaid orders older than 30 minutes and returns their ids.
func (r *Repo) ExpirePending(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, expirePendingSQL)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
