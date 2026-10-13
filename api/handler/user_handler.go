package handler

import (
	"errors"
	"log"
	"net/http"

	"movie-reservation-system/internal/service"
	"movie-reservation-system/internal/util"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service service.UserService
}

func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) GetMe(ctx *gin.Context) {
	userID, ok := ctx.Get(util.CtxUserID)
	id, isInt := userID.(int64)
	if !ok || !isInt {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.service.GetByID(ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		log.Printf("get user failed: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, user)
}
