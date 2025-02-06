package main

import (
	"log"
	"net/http"
	"os"

	"github.com/geoo115/E-commerceMicroservices/api-gateway/router"
	"github.com/spf13/viper"
)

func initConfig() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Println("CONFIG_PATH not set, using default config path: config.yaml")
		configPath = "config.yaml"
	}
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}
}

func main() {
	initConfig()
	port := viper.GetString("server.port")
	if port == "" {
		port = "8080"
	}
	r := router.SetupRouter()
	log.Printf("API Gateway running on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
