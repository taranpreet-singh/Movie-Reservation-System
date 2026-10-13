package routes

import (
	"movie-reservation-system/api/handler"

	"github.com/gin-gonic/gin"
)

func NewAuthRoutes(rg *gin.RouterGroup, h *handler.AuthHandler) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
	}
}
