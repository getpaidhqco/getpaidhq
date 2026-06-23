package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvoice_ApplyDiscountTotals(t *testing.T) {
	inv := Invoice{LineItems: []InvoiceLineItem{{Id: "l1", Total: 1000}, {Id: "l2", Total: 500}}}
	inv.recalculate()
	inv.ApplyDiscountTotals(map[string]int64{"l1": 250})
	assert.EqualValues(t, 250, inv.DiscountTotal)
	assert.EqualValues(t, 1250, inv.Total) // 1500 - 250
}
