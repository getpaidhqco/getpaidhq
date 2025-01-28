package main

import (
	"payloop/internal/application/bootstrap"
)

func main() {
	err := bootstrap.RootApp.Execute()
	if err != nil {
		panic(err)
	}
}
