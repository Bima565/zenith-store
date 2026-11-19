package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"zenith-store/backend/services"
)

type GameHandler struct {
	vipayment *services.VIPaymentService
}

func NewGameHandler(vipayment *services.VIPaymentService) *GameHandler {
	return &GameHandler{vipayment: vipayment}
}

func (h *GameHandler) GetServices(c *gin.Context) {
	services, err := h.vipayment.GetServices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": services})
}
