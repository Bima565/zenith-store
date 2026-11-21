package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                     string
	CORSOrigins              []string
	VIPaymentAPIID           string
	VIPaymentAPIKey          string
	VIPaymentBaseURL         string
	VIPaymentServicesPath    string
	VIPaymentNicknamePath    string
	VIPaymentTransactionPath string
	SteamGridAPIKey          string
}

func Load() (*Config, error) {
	if err := godotenv.Load(".env"); err != nil {
		_ = godotenv.Load("src/backend/.env")
	}

	origins := strings.Split(os.Getenv("CORS_ORIGINS"), ",")
	if strings.TrimSpace(os.Getenv("CORS_ORIGINS")) == "" {
		origins = []string{"http://localhost:4200"}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	baseURL := os.Getenv("VIPAYMENT_BASE_URL")
	if baseURL == "" {
		baseURL = "https://vip-reseller.co.id"
	}

	apiKey := os.Getenv("VIPAYMENT_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("VIPAYMENT_SIGN")
	}

	steamGridKey := os.Getenv("STEAMGRID")
	if steamGridKey == "" {
		steamGridKey = os.Getenv("STEAMGRID_API_KEY")
	}
	if steamGridKey == "" {
		steamGridKey = os.Getenv("STEAMGRIDDB_API_KEY")
	}

	return &Config{
		Port:                     port,
		CORSOrigins:              origins,
		VIPaymentAPIID:           os.Getenv("VIPAYMENT_API_ID"),
		VIPaymentAPIKey:          apiKey,
		VIPaymentBaseURL:         strings.TrimRight(baseURL, "/"),
		VIPaymentServicesPath:    envOrDefault("VIPAYMENT_SERVICES_PATH", "/api/game-feature"),
		VIPaymentNicknamePath:    envOrDefault("VIPAYMENT_NICKNAME_PATH", "/api/game-feature"),
		VIPaymentTransactionPath: envOrDefault("VIPAYMENT_TRANSACTION_PATH", "/api/game-feature"),
		SteamGridAPIKey:          strings.TrimSpace(steamGridKey),
	}, nil
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
