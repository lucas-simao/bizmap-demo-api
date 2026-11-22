package main

import "fmt"

// Order lifecycle: pending → paid → shipped, and pending|paid → cancelled.
// pending orders that are never paid expire (see ExpireJob).
var allowedTransitions = map[string][]string{
	"pending": {"paid", "cancelled", "expired"},
	"paid":    {"shipped", "cancelled"},
}

// CanTransition reports whether an order may move from one status to another.
func CanTransition(from, to string) error {
	for _, s := range allowedTransitions[from] {
		if s == to {
			return nil
		}
	}
	return fmt.Errorf("order cannot go from %s to %s", from, to)
}
