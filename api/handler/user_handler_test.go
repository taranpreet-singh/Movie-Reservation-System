package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"movie-reservation-system/internal/model"
	"movie-reservation-system/internal/util"

	"github.com/gin-gonic/gin"
)

type fakeUserService struct {
	getFn func(ctx context.Context, id int64) (*model.User, error)
	calls int
}

func (f *fakeUserService) GetByID(ctx context.Context, id int64) (*model.User, error) {
	f.calls++
	return f.getFn(ctx, id)
}

func userRouter(svc *fakeUserService, contextUserID any) *gin.Engine {
	h := NewUserHandler(svc)
	r := gin.New()
	r.GET("/me", func(ctx *gin.Context) {
		if contextUserID != nil {
			ctx.Set(util.CtxUserID, contextUserID)
		}
	}, h.GetMe)
	return r
}

func getMe(r http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGetMe_Success(t *testing.T) {
	var gotID int64
	svc := &fakeUserService{
		getFn: func(_ context.Context, id int64) (*model.User, error) {
			gotID = id
			return &model.User{ID: id, Name: "Alice", Email: "alice@example.com", PasswordHash: "secret-hash", Role: model.RoleUser}, nil
		},
	}

	w := getMe(userRouter(svc, int64(5)))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if gotID != 5 {
		t.Errorf("service called with id %d, want 5", gotID)
	}
	if body := decodeBody(t, w); body["email"] != "alice@example.com" {
		t.Errorf("email = %v, want alice@example.com", body["email"])
	}
	if strings.Contains(w.Body.String(), "secret-hash") {
		t.Error("response leaks the password hash")
	}
}

func TestGetMe_UnauthorizedWhenContextHasNoValidUserID(t *testing.T) {
	tests := []struct {
		name          string
		contextUserID any
	}{
		{"user id missing", nil},
		{"user id of the wrong type", 5},
		{"user id as a string", "5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeUserService{}

			w := getMe(userRouter(svc, tt.contextUserID))

			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
			}
			if svc.calls != 0 {
				t.Error("the service must not be called without a valid user id")
			}
		})
	}
}

func TestGetMe_ErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"user no longer exists", util.ErrNotFound, http.StatusNotFound},
		{"unexpected failure", errors.New("pq: connection refused"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeUserService{
				getFn: func(context.Context, int64) (*model.User, error) { return nil, tt.err },
			}

			w := getMe(userRouter(svc, int64(5)))

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusInternalServerError && strings.Contains(w.Body.String(), "connection refused") {
				t.Error("a 500 response must not leak the underlying error")
			}
		})
	}
}
