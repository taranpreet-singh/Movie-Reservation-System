package service

import (
	"context"

	"movie-reservation-system/internal/model"
	"movie-reservation-system/internal/repository"
)

type UserService interface {
	GetByID(ctx context.Context, id int64) (*model.User, error)
}

type userService struct {
	users repository.UserRepository
}

func NewUserService(users repository.UserRepository) UserService {
	return &userService{users: users}
}

func (s *userService) GetByID(ctx context.Context, id int64) (*model.User, error) {
	return s.users.GetByID(ctx, id)
}
