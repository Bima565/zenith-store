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
		ReferenceID:    generateReferenceID(),
		ServiceCode:    request.ServiceCode,
		UserID:         request.UserID,
		ZoneID:         request.ZoneID,
		AdditionalData: request.AdditionalData,
		Quantity:       request.Quantity,
		Status:         "pending",
	}

	transaction, err := h.vipayment.CreateTransaction(c.Request.Context(), order)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		return
	}

	order.ReferenceID = transaction.TrxID
	order.Status = transaction.Status
	order.Note = "Checkout has been submitted"

	h.mu.Lock()
	h.orders[order.ReferenceID] = order
	h.mu.Unlock()

	c.JSON(http.StatusCreated, gin.H{
		"data":     order,
		"provider": transaction,
	})
}

func (h *OrderHandler) GetStatus(c *gin.Context) {
	trxID := c.Param("trxid")
	if trxID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "trxid wajib diisi"})
		return
	}

	transactions, err := h.vipayment.GetTransactionStatus(c.Request.Context(), trxID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": transactions})
}

func (h *OrderHandler) Webhook(c *gin.Context) {
	if !h.vipayment.ValidateWebhookSignature(c.GetHeader("X-Client-Signature")) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "signature webhook tidak valid"})
		return
	}

	var payload models.WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "payload webhook tidak valid", "error": err.Error()})
		return
	}

	h.mu.Lock()
	if order, ok := h.orders[payload.Transaction.TrxID]; ok {
		order.Status = payload.Transaction.Status
		order.Note = payload.Transaction.Note
		h.orders[payload.Transaction.TrxID] = order
	}
	h.mu.Unlock()
	c.JSON(http.StatusOK, gin.H{"message": "webhook diterima"})
}

func generateReferenceID() string {
	return "ZNS-" + time.Now().UTC().Format("20060102150405") + "-" + time.Now().Format("000000")
}
