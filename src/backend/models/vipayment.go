package models

type VIPaymentResponse struct {
	Result  bool        `json:"result"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type ServicePrice struct {
	Basic   int64 `json:"basic"`
	Premium int64 `json:"premium"`
	Special int64 `json:"special"`
}

type GameService struct {
	Code        string       `json:"code"`
	Game        string       `json:"game"`
	Name        string       `json:"name"`
	Price       ServicePrice `json:"price"`
	Description string       `json:"description"`
	Server      string       `json:"server"`
	Status      string       `json:"status"`
	Category    string       `json:"category,omitempty"`
	Icon        string       `json:"icon"`
	Grid        string       `json:"grid,omitempty"`
}

type NicknameResponse struct {
	Nickname string   `json:"nickname"`
	Country  *Country `json:"country,omitempty"`
}

type Country struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Transaction struct {
	TrxID          string `json:"trxid"`
	Data           string `json:"data"`
	Zone           string `json:"zone"`
	Service        string `json:"service"`
	Status         string `json:"status"`
	Note           string `json:"note"`
	Price          int64  `json:"price"`
	Balance        int64  `json:"balance,omitempty"`
	AdditionalData string `json:"additional_data,omitempty"`
}

type WebhookPayload struct {
	Transaction Transaction `json:"data"`
}
