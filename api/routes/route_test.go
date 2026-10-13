package routes_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"movie-reservation-system/api/handler"
	"movie-reservation-system/api/middleware"
	"movie-reservation-system/api/routes"
	"movie-reservation-system/internal/model"
	"movie-reservation-system/internal/util"

	"github.com/gin-gonic/gin"
)

const testSecret = "test-secret"

type stubAuthService struct{}

func (stubAuthService) Register(_ context.Context, name, email, _ string) (*model.User, error) {
	return &model.User{ID: 1, Name: name, Email: email, Role: model.RoleUser}, nil
}

func (stubAuthService) Login(context.Context, string, string) (string, error) {
	return "token", nil
}

type stubUserService struct{}

func (stubUserService) GetByID(_ context.Context, id int64) (*model.User, error) {
	return &model.User{ID: id, Name: "Alice", Email: "alice@example.com", Role: model.RoleUser}, nil
}

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return routes.New(
		handler.NewAuthHandler(stubAuthService{}),
		handler.NewUserHandler(stubUserService{}),
		middleware.AuthRequired(testSecret),
	)
}

func do(r http.Handler, method, path, body, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHealthz(t *testing.T) {
	w := do(newTestRouter(), http.MethodGet, "/healthz", "", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthRoutes_ArePublicAndVersioned(t *testing.T) {
	r := newTestRouter()

	register := do(r, http.MethodPost, "/api/v1/auth/register", `{"name":"Alice","email":"alice@example.com","password":"supersecret"}`, "")
	if register.Code != http.StatusCreated {
		t.Errorf("register status = %d, want %d", register.Code, http.StatusCreated)
	}

	login := do(r, http.MethodPost, "/api/v1/auth/login", `{"email":"alice@example.com","password":"supersecret"}`, "")
	if login.Code != http.StatusOK {
		t.Errorf("login status = %d, want %d", login.Code, http.StatusOK)
	}
}

func TestUnversionedAuthPathsDoNotExist(t *testing.T) {
	w := do(newTestRouter(), http.MethodPost, "/auth/login", `{"email":"alice@example.com","password":"supersecret"}`, "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUserMe_RequiresAuthentication(t *testing.T) {
	r := newTestRouter()

	validToken, err := util.GenerateToken(1, "user", testSecret)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	foreignToken, err := util.GenerateToken(1, "user", "another-secret")
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	tests := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{"no token", "", http.StatusUnauthorized},
		{"garbage token", "Bearer nope", http.StatusUnauthorized},
		{"token signed with another secret", "Bearer " + foreignToken, http.StatusUnauthorized},
		{"valid token", "Bearer " + validToken, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := do(r, http.MethodGet, "/api/v1/user/me", "", tt.header)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK && !strings.Contains(w.Body.String(), "alice@example.com") {
				t.Errorf("body does not contain the user: %s", w.Body.String())
			}
		})
	}
}
