package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"zenith-store/backend/services"
)

type GameHandler struct {
	vipayment *services.VIPaymentService
	steamgrid *services.SteamGridService
}

func NewGameHandler(vipayment *services.VIPaymentService, steamgrid *services.SteamGridService) *GameHandler {
	return &GameHandler{
		vipayment: vipayment,
		steamgrid: steamgrid,
	}
}

func (h *GameHandler) GetServices(c *gin.Context) {
	list, err := h.vipayment.GetServices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		return
	}
	if h.steamgrid != nil {
		list = h.steamgrid.EnrichServices(c.Request.Context(), list, 16)
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}
