package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"io"
	"net/http"
	"payloop/internal/api/middlewares"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/infrastructure/db/postgres"
	"payloop/internal/lib"
	"testing"
)

func TestCreateOrder(t *testing.T) {
	ctx := context.Background()
	logger := lib.GetLogger()

	app := fx.New(fx.Options(
		lib.Module,
		Module,
		middlewares.Module,
		postgres.Module,
	), fx.Options(
		fx.WithLogger(func() fxevent.Logger {
			return lib.GetFxLogger()
		}),
		fx.Invoke(func(orderService OrderService) {
			logger.Info("Starting application")

			_, err := orderService.CreateOrder(ctx, orders.CreateOrderInput{
				OrgId:    "org_2syb0uTnhuKtQTaLO6EAk1iIUnu",
				Currency: "ZAR",
				Customer: orders.CreateOrderCommandCustomer{
					Id: "cus_2u7124uRNWnn2NpQdpSa6b1kLqC",
				},
				PaymentMethodId: "pm_2u718M3todYa5mkGPM9JpCWWhw2",
				CartItems: []orders.CartItem{
					{ProductId: "prod-1", PriceId: "cyc-1", Quantity: 1},
				},
				PspId:    "Paystack",
				Metadata: nil,
			})
			assert.Equal(t, err, nil)
		}),
	))
	app.Start(ctx)
	defer func() {
		app.Stop(ctx)
	}()

}

func TestCreateOrders(t *testing.T) {
	url1 := "http://localhost:8888/api/orders"
	apiKey := "org_2syb0uTnhuKtQTaLO6EAk1iIUnu"
	jsonData1 := []byte(`{
    "psp_id": "Paystack",
    "payment_method_id": "pm_2u718M3todYa5mkGPM9JpCWWhw2",
    "customer": {
       "id": "cus_2u7124uRNWnn2NpQdpSa6b1kLqC"
    },
    "cart":{
		"currency": "ZAR",
        "items": [{
            "product_id": "prod-1",
            "price_id": "cyc-1",
            "quantity": 1
        }]
    }
}`)

	for i := 0; i < 10; i++ {
		// First POST request
		req1, err := http.NewRequest("POST", url1, bytes.NewBuffer(jsonData1))
		if err != nil {
			fmt.Printf("Error creating request to URL1: %v\n", err)
			return
		}
		req1.Header.Set("Content-Type", "application/json")
		req1.Header.Set("X-API-Key", apiKey)

		client := &http.Client{}
		resp1, err := client.Do(req1)
		if err != nil {
			fmt.Printf("Error making POST request to URL1: %v\n", err)
			return
		}
		defer resp1.Body.Close()
		body1, err := io.ReadAll(resp1.Body)
		if err != nil {
			fmt.Printf("Error reading response from URL1: %v\n", err)
			return
		}
		fmt.Printf("Response from URL1: %s\n", string(body1))

		// get the order id
		var result map[string]interface{}
		if err := json.Unmarshal(body1, &result); err != nil {
			fmt.Printf("Error parsing response from URL1: %v\n", err)
			return
		}
		orderID, ok := result["order"].(map[string]interface{})["id"].(string)
		if !ok {
			fmt.Println("Order ID not found in response")
			return
		}
		fmt.Printf("Order ID from URL1: %s\n", orderID)

		// Complete the order
		url2 := fmt.Sprintf("http://localhost:8888/api/orders/%s/complete", orderID)
		req2, err := http.NewRequest("POST", url2, nil)
		if err != nil {
			fmt.Printf("Error creating request to URL2: %v\n", err)
			return
		}
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("X-API-Key", apiKey)
		resp2, err := client.Do(req2)
		if err != nil {
			fmt.Printf("Error making POST request to URL2: %v\n", err)
			return
		}
		defer resp2.Body.Close()
		_, err = io.ReadAll(resp2.Body)
		if err != nil {
			fmt.Printf("Error reading response from URL2: %v\n", err)
			return
		}
		fmt.Printf("-------- done %d\n", i)
	}

}
