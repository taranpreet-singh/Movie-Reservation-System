package service

import (
	"context"
	"errors"
	"strings"

	"movie-reservation-system/internal/model"
	"movie-reservation-system/internal/repository"
	"movie-reservation-system/internal/util"
)

type AuthService interface {
	Register(ctx context.Context, name, email, password string) (*model.User, error)
	Login(ctx context.Context, email, password string) (string, error)
}

type authService struct {
	users     repository.UserRepository
	jwtSecret string
	dummyHash string
}

func NewAuthService(users repository.UserRepository, jwtSecret string) AuthService {
	dummy, _ := util.HashPassword("timing-equalizer")
	return &authService{users: users, jwtSecret: jwtSecret, dummyHash: dummy}
}

func (s *authService) Register(ctx context.Context, name, email, password string) (*model.User, error) {
	hash, err := util.HashPassword(password)
	if err != nil {
		return nil, err
	}

	u := &model.User{
		Name:         strings.TrimSpace(name),
		Email:        strings.ToLower(strings.TrimSpace(email)),
		PasswordHash: hash,
		Role:         model.RoleUser,
	}

	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (string, error) {
	u, err := s.users.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			_ = util.CheckPassword(s.dummyHash, password)
			return "", util.ErrInvalidCredentials
		}
		return "", err
	}

	if err := util.CheckPassword(u.PasswordHash, password); err != nil {
		return "", util.ErrInvalidCredentials
	}

	return util.GenerateToken(u.ID, string(u.Role), s.jwtSecret)
}
