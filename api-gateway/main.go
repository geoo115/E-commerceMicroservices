package main

import (
	"log"
	"net/http"

	"api-gateway/router"

	"github.com/spf13/viper"
)

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
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
