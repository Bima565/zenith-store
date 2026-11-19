package services

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"zenith-store/backend/config"
	"zenith-store/backend/models"
)

type VIPaymentService struct {
	config     *config.Config
	httpClient *http.Client
}

func NewVIPaymentService(cfg *config.Config) *VIPaymentService {
	return &VIPaymentService{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *VIPaymentService) GenerateSign(parts ...string) string {
	values := append([]string{s.config.VIPaymentAPIID, s.config.VIPaymentAPIKey}, parts...)
	sum := md5.Sum([]byte(strings.Join(values, "")))
	return hex.EncodeToString(sum[:])
}

func (s *VIPaymentService) GetServices(ctx context.Context) ([]models.GameService, error) {
	if s.config.VIPaymentAPIID == "" || s.config.VIPaymentAPIKey == "" {
		return demoServices(), nil
	}

	endpoint := s.config.VIPaymentBaseURL + s.config.VIPaymentServicesPath
	params := url.Values{}
	params.Set("api_id", s.config.VIPaymentAPIID)
	params.Set("sign", s.GenerateSign("services"))

	var raw models.VIPaymentResponse
	if err := s.request(ctx, http.MethodGet, endpoint, params, &raw); err != nil {
		return nil, err
	}

	data, err := json.Marshal(raw.Data)
	if err != nil {
		return nil, err
	}

	var services []models.GameService
	if err := json.Unmarshal(data, &services); err != nil {
		return nil, fmt.Errorf("decode VIPayment services: %w", err)
	}
	return services, nil
}

func (s *VIPaymentService) GetNickname(ctx context.Context, gameCode, userID, zoneID string) (*models.NicknameResponse, error) {
	if s.config.VIPaymentAPIID == "" || s.config.VIPaymentAPIKey == "" {
		return &models.NicknameResponse{
			Nickname: "Demo User",
			UserID:   userID,
			ZoneID:   zoneID,
		}, nil
	}

	params := url.Values{}
	params.Set("api_id", s.config.VIPaymentAPIID)
	params.Set("sign", s.GenerateSign(gameCode, userID, zoneID))
	params.Set("game_code", gameCode)
	params.Set("user_id", userID)
	params.Set("zone_id", zoneID)

	endpoint := s.config.VIPaymentBaseURL + s.config.VIPaymentNicknamePath
	var result struct {
		Data models.NicknameResponse `json:"data"`
	}
	if err := s.request(ctx, http.MethodPost, endpoint, params, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (s *VIPaymentService) CreateTransaction(ctx context.Context, order models.Order) (map[string]interface{}, error) {
	if s.config.VIPaymentAPIID == "" || s.config.VIPaymentAPIKey == "" {
		return map[string]interface{}{
			"status": "demo",
			"message": "Transaction recorded in demo mode",
		}, nil
	}

	params := url.Values{}
	params.Set("api_id", s.config.VIPaymentAPIID)
	params.Set("sign", s.GenerateSign(order.ReferenceID, order.ServiceCode, order.UserID))
	params.Set("ref_id", order.ReferenceID)
	params.Set("service_code", order.ServiceCode)
	params.Set("user_id", order.UserID)
	params.Set("zone_id", order.ZoneID)
	params.Set("quantity", fmt.Sprintf("%d", order.Quantity))

	endpoint := s.config.VIPaymentBaseURL + s.config.VIPaymentTransactionPath
	var result map[string]interface{}
	if err := s.request(ctx, http.MethodPost, endpoint, params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *VIPaymentService) request(
	ctx context.Context,
	method string,
	endpoint string,
	params url.Values,
	target interface{},
) error {
	requestURL := endpoint
	if method == http.MethodGet {
		requestURL += "?" + params.Encode()
	}

	var requestBody *strings.Reader
	if method == http.MethodPost {
		requestBody = strings.NewReader(params.Encode())
	} else {
		requestBody = strings.NewReader("")
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, requestBody)
	if err != nil {
		return err
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("VIPayment request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("VIPayment returned status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode VIPayment response: %w", err)
	}
	return nil
}

func demoServices() []models.GameService {
	return []models.GameService{
		{Code: "MLBB-DIAMOND", Name: "Mobile Legends", Category: "Game", Description: "Diamond Mobile Legends", Price: 10000},
		{Code: "FF-DIAMOND", Name: "Free Fire", Category: "Game", Description: "Diamond Free Fire", Price: 12000},
		{Code: "PUBG-UC", Name: "PUBG Mobile", Category: "Game", Description: "Unknown Cash", Price: 15000},
	}
}
