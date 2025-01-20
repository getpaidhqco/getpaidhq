package main

import (
	"go.uber.org/fx"
	stdhttp "net/http"
	"payloop/internal/db"
	"payloop/internal/http"
	"payloop/internal/orders"
	"payloop/internal/repositories"
	"payloop/internal/services"
)

func main() {
	app := fx.New(
		http.Module,
		db.Module,
		services.Module,
		repositories.Module,
		orders.Module,
		fx.Invoke(func(handler stdhttp.Handler) error {
			go func() {
				err := stdhttp.ListenAndServe(":8080", handler)
				if err != nil {
					panic(err)
				}
			}()
			return nil
		}),
	)

	app.Run()
}
