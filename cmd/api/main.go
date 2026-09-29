package main

import (
	"movie-reservation-system/internal/config"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	config.Load()

	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"health": "Okay"})
	})

	router.Run()
}
