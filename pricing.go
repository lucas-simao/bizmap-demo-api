package main

// Discount rules. Kept in one place so a pricing change shows up in BizMap
// as "this function reaches POST /api/orders and writes orders.discount_cents".
const (
	bulkQtyThreshold = 10 // units in one order
	bulkDiscountPct  = 5
	goldDiscountPct  = 10 // loyalty tier "gold"
	maxDiscountPct   = 15
)

// DiscountCents returns the discount for an order subtotal: gold customers get
// 10%, orders of 10+ units get 5% more, capped at 15%.
func DiscountCents(subtotalCents int64, units int, tier string) int64 {
	pct := 0
	if tier == "gold" {
		pct += goldDiscountPct
	}
	if units >= bulkQtyThreshold {
		pct += bulkDiscountPct
	}
	if pct > maxDiscountPct {
		pct = maxDiscountPct
	}
	return subtotalCents * int64(pct) / 100
}
