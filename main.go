package main

import (
	"payloop/internal/config"
)

func main() {
	app, err := config.NewApp()
	if err != nil {
		panic(err)
	}
	if err := app.Run(); err != nil {
		panic(err)
	}
}
