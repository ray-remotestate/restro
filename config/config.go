package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

var SecretKey []byte

func Init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("%v", err)
	}

	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		log.Fatal("JWT secret key not set")
	}
	SecretKey = []byte(secret)
}