package env

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"path/filepath"
)

func Load() {
	pwd, err := os.Getwd()
	envFile := filepath.Join(pwd, "../../", ".env")

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = godotenv.Load(envFile)
}
