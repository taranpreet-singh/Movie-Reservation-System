package main

import (
	"log"
	"movie-reservation-system/api/handler"
	"movie-reservation-system/api/middleware"
	"movie-reservation-system/api/routes"
	"movie-reservation-system/internal/config"
	"movie-reservation-system/internal/db"
	"movie-reservation-system/internal/repository"
	"movie-reservation-system/internal/service"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" || cfg.JWTSecret == "" {
		log.Fatal("DATABASE_URL and JWT_SECRET must be set")
	}

	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	userRepo := repository.NewUserRepository(database)
	authService := service.NewAuthService(userRepo, cfg.JWTSecret)
	authHandler := handler.NewAuthHandler(authService)

	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)

	router := routes.New(authHandler, userHandler, middleware.AuthRequired(cfg.JWTSecret))
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
