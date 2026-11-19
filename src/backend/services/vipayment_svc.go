package services

import (
	"context"
	"crypto/md5"
	"crypto/subtle"
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
		config:     cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *VIPaymentService) GenerateSign() string {
	sum := md5.Sum([]byte(s.config.VIPaymentAPIID + s.config.VIPaymentAPIKey))
	return hex.EncodeToString(sum[:])
}

func (s *VIPaymentService) GetServices(ctx context.Context) ([]models.GameService, error) {
	if s.demoMode() {
		return demoServices(), nil
	}

	var response struct {
		Result  bool                 `json:"result"`
		Message string               `json:"message"`
		Data    []models.GameService `json:"data"`
	}
	params := url.Values{"type": {"services"}}
	if err := s.request(ctx, params, &response); err != nil {
		return nil, err
	}
	if !response.Result {
		return nil, fmt.Errorf("VIPayment services: %s", response.Message)
	}
	return response.Data, nil
}

func (s *VIPaymentService) GetNickname(ctx context.Context, gameCode, userID, zoneID string) (*models.NicknameResponse, error) {
	if s.demoMode() {
		return &models.NicknameResponse{Nickname: "Demo User"}, nil
	}

	var response struct {
		Result  bool            `json:"result"`
		Message string          `json:"message"`
		Data    string          `json:"data"`
		Country *models.Country `json:"country,omitempty"`
	}
	params := url.Values{
		"type":              {"get-nickname"},
		"code":              {gameCode},
		"target":            {userID},
		"additional_target": {zoneID},
	}
	if err := s.request(ctx, params, &response); err != nil {
		return nil, err
	}
	if !response.Result {
		return nil, fmt.Errorf("VIPayment nickname: %s", response.Message)
	}
	return &models.NicknameResponse{Nickname: response.Data, Country: response.Country}, nil
}

func (s *VIPaymentService) CreateTransaction(ctx context.Context, order models.Order) (*models.Transaction, error) {
	if s.demoMode() {
		return &models.Transaction{
			TrxID:  order.ReferenceID,
			Data:   order.UserID,
			Zone:   order.ZoneID,
			Status: "waiting",
		}, nil
	}

	var response struct {
		Result  bool               `json:"result"`
		Message string             `json:"message"`
		Data    models.Transaction `json:"data"`
	}
	params := url.Values{
		"type":                 {"order"},
		"service":              {order.ServiceCode},
		"data_no":              {order.UserID},
		"data_zone":            {order.ZoneID},
		"post_additional_data": {order.AdditionalData},
	}
	if err := s.request(ctx, params, &response); err != nil {
		return nil, err
	}
	if !response.Result {
		return nil, fmt.Errorf("VIPayment order: %s", response.Message)
	}
	return &response.Data, nil
}

func (s *VIPaymentService) GetTransactionStatus(ctx context.Context, trxID string) ([]models.Transaction, error) {
	if s.demoMode() {
		return []models.Transaction{{TrxID: trxID, Status: "waiting"}}, nil
	}

	var response struct {
		Result  bool                 `json:"result"`
		Message string               `json:"message"`
		Data    []models.Transaction `json:"data"`
	}
	params := url.Values{"type": {"status"}, "trxid": {trxID}}
	if err := s.request(ctx, params, &response); err != nil {
		return nil, err
	}
	if !response.Result {
		return nil, fmt.Errorf("VIPayment status: %s", response.Message)
	}
	return response.Data, nil
}

func (s *VIPaymentService) ValidateWebhookSignature(signature string) bool {
	expected := s.GenerateSign()
	provided, err := hex.DecodeString(strings.TrimSpace(signature))
	if err != nil || len(provided) != len(expected)/2 {
		return false
	}
	expectedBytes, _ := hex.DecodeString(expected)
	return subtle.ConstantTimeCompare(provided, expectedBytes) == 1
}

func (s *VIPaymentService) request(ctx context.Context, params url.Values, target interface{}) error {
	params.Set("key", s.config.VIPaymentAPIKey)
	params.Set("sign", s.GenerateSign())
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.config.VIPaymentBaseURL+s.config.VIPaymentServicesPath,
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := s.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("VIPayment request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("VIPayment returned status %d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode VIPayment response: %w", err)
	}
	return nil
}

func (s *VIPaymentService) demoMode() bool {
	return s.config.VIPaymentAPIID == "" || s.config.VIPaymentAPIKey == ""
}

func demoServices() []models.GameService {
	return []models.GameService{
		{Code: "MLBB-DIAMOND", Game: "Mobile Legends", Name: "Diamond Mobile Legends", Price: models.ServicePrice{Basic: 10000}, Status: "available"},
		{Code: "FF-DIAMOND", Game: "Free Fire", Name: "Diamond Free Fire", Price: models.ServicePrice{Basic: 12000}, Status: "available"},
		{Code: "PUBG-UC", Game: "PUBG Mobile", Name: "Unknown Cash", Price: models.ServicePrice{Basic: 15000}, Status: "available"},
	}
}
