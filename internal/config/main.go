package config

// Start initializes the application and runs the HTTP server.
func Start() {
	app, err := NewApp()
	if err != nil {
		panic(err)
	}
	if err := app.Run(); err != nil {
		panic(err)
	}
}
