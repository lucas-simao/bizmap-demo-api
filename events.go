package main

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
)

// OrderEvent is one entry of the order timeline in the Firestore collection
// "order_events". The API writes it on every status change; the web app
// shows it live and its Cloud Function sends notifications from it.
type OrderEvent struct {
	OrderID   string    `firestore:"orderId"`
	Type      string    `firestore:"type"`
	Status    string    `firestore:"status"`
	Source    string    `firestore:"source"`
	CreatedAt time.Time `firestore:"createdAt"`
}

type Events struct {
	fs *firestore.Client
}

func NewEvents(fs *firestore.Client) *Events { return &Events{fs: fs} }

// Publish appends an event to the shared order timeline.
func (e *Events) Publish(ctx context.Context, orderID, typ, status string) error {
	_, _, err := e.fs.Collection("order_events").Add(ctx, OrderEvent{
		OrderID: orderID, Type: typ, Status: status, Source: "api", CreatedAt: time.Now(),
	})
	return err
}
