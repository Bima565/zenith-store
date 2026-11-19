package models

type VIPaymentResponse struct {
	Status  interface{} `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type GameService struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
}

type NicknameResponse struct {
	Nickname string `json:"nickname"`
	UserID   string `json:"user_id"`
	ZoneID   string `json:"zone_id,omitempty"`
}
