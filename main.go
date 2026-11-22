// Command api serves the Northwind Outfitters shop API: customers, products,
// orders with discounts and stock reservation, payments and shipping.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"cloud.google.com/go/firestore"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	fs, err := firestore.NewClient(ctx, "demo-shop")
	if err != nil {
		log.Fatal(err)
	}
	defer fs.Close()

	repo := NewRepo(pool)
	events := NewEvents(fs)
	h := NewHandler(repo, events)

	e := echo.New()
	api := e.Group("/api")
	registerRoutes(api, h)

	go NewExpireJob(repo, events).Run(ctx)

	if err := e.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
