package routes

import (
	"net/http"

	"movie-reservation-system/api/handler"

	"github.com/gin-gonic/gin"
)

func New(authHandler *handler.AuthHandler, userHandler *handler.UserHandler, authRequired gin.HandlerFunc) *gin.Engine {
	router := gin.Default()

	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"health": "Okay"})
	})

	v1 := router.Group("/api/v1")
	NewAuthRoutes(v1, authHandler)
	NewUserRoutes(v1, userHandler, authRequired)
	return router
}
