package main

import (
	"payloop/internal/bootstrap"
)

func main() {
	err := bootstrap.RootApp.Execute()
	if err != nil {
		panic(err)
	}
}
