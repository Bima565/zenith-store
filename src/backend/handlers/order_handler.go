package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"zenith-store/backend/models"
	"zenith-store/backend/services"
)

type OrderHandler struct {
	vipayment *services.VIPaymentService
	mu        sync.RWMutex
	orders    map[string]models.Order
}

func NewOrderHandler(vipayment *services.VIPaymentService) *OrderHandler {
	return &OrderHandler{
		vipayment: vipayment,
		orders:    make(map[string]models.Order),
	}
}

func (h *OrderHandler) GetNickname(c *gin.Context) {
	gameCode := c.Query("game_code")
	userID := c.Query("user_id")
	zoneID := c.Query("zone_id")
	if gameCode == "" || userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "game_code dan user_id wajib diisi"})
		return
	}

	nickname, err := h.vipayment.GetNickname(c.Request.Context(), gameCode, userID, zoneID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": nickname})
}

func (h *OrderHandler) Checkout(c *gin.Context) {
	var request models.CheckoutRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Request tidak valid", "error": err.Error()})
		return
	}
	if request.Quantity < 1 {
		request.Quantity = 1
	}

	order := models.Order{
		ReferenceID: generateReferenceID(),
		ServiceCode: request.ServiceCode,
		UserID:      request.UserID,
		ZoneID:      request.ZoneID,
		Quantity:    request.Quantity,
		Status:      "pending",
	}

	transaction, err := h.vipayment.CreateTransaction(c.Request.Context(), order)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		return
	}

	if status, ok := transaction["status"].(string); ok && status != "" {
		order.Status = status
	}
	order.Note = "Checkout has been submitted"

	h.mu.Lock()
	h.orders[order.ReferenceID] = order
	h.mu.Unlock()

	c.JSON(http.StatusCreated, gin.H{
		"data":        order,
		"provider":    transaction,
	})
}

func generateReferenceID() string {
	return "ZNS-" + time.Now().UTC().Format("20060102150405") + "-" + time.Now().Format("000000")
}
