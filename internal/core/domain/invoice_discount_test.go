package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvoiceRecalculateAppliesDiscount(t *testing.T) {
	inv := &Invoice{OrgId: "org_1", Id: "inv_1"}
	inv.LineItems = []InvoiceLineItem{
		{Id: "l1", Total: 1000, DiscountTotal: 250},
		{Id: "l2", Total: 500, DiscountTotal: 0},
	}
	inv.recalculate()
	assert.EqualValues(t, 1500, inv.Subtotal)
	assert.EqualValues(t, 250, inv.DiscountTotal)
	assert.EqualValues(t, 1250, inv.Total)
}
