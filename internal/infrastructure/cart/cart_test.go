package cart

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAddItem(t *testing.T) {
	cart := New(CartOptions{Currency: "USD"})
	item := Item{
		Id:        "1",
		ProductId: "prod_1",
		Price: Price{
			Id:                 "price_1",
			Category:           "subscription",
			Scheme:             "fixed",
			Currency:           "USD",
			UnitPrice:          1000,
			MinPrice:           0,
			SuggestedPrice:     0,
			BillingInterval:    "month",
			BillingIntervalQty: 1,
			TrialInterval:      "none",
			TrialIntervalQty:   0,
			TaxCode:            "exempt",
		},
		Description: "Test Item",
		Quantity:    1,
	}
	cart.AddItem(item)
	assert.Equal(t, cart.Total, 1000, "they should be equal")

	_, err := cart.AdjustQuantity("1", 2)
	assert.Nil(t, err)
	assert.Equal(t, cart.Total, 2000, "they should be equal")

}
