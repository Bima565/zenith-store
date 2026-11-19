package models

type CheckoutRequest struct {
	ServiceCode string `json:"service_code" binding:"required"`
	UserID      string `json:"user_id" binding:"required"`
	ZoneID      string `json:"zone_id"`
	Quantity    int    `json:"quantity"`
}

type Order struct {
	ReferenceID string `json:"reference_id"`
	ServiceCode string `json:"service_code"`
	UserID      string `json:"user_id"`
	ZoneID      string `json:"zone_id,omitempty"`
	Quantity    int    `json:"quantity"`
	Status      string `json:"status"`
	Note        string `json:"note,omitempty"`
}
