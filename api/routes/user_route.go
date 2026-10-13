package routes

import (
	"movie-reservation-system/api/handler"

	"github.com/gin-gonic/gin"
)

func NewUserRoutes(rg *gin.RouterGroup, h *handler.UserHandler, authRequired gin.HandlerFunc) {
	user := rg.Group("/user", authRequired)
	{
		user.GET("/me", h.GetMe)
	}
}
