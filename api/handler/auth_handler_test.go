package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"movie-reservation-system/internal/model"
	"movie-reservation-system/internal/util"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeAuthService struct {
	registerFn func(ctx context.Context, name, email, password string) (*model.User, error)
	loginFn    func(ctx context.Context, email, password string) (string, error)
	calls      int
}

func (f *fakeAuthService) Register(ctx context.Context, name, email, password string) (*model.User, error) {
	f.calls++
	return f.registerFn(ctx, name, email, password)
}

func (f *fakeAuthService) Login(ctx context.Context, email, password string) (string, error) {
	f.calls++
	return f.loginFn(ctx, email, password)
}

func authRouter(svc *fakeAuthService) *gin.Engine {
	h := NewAuthHandler(svc)
	r := gin.New()
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	return r
}

func postJSON(r http.Handler, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v (body: %s)", err, w.Body.String())
	}
	return body
}

func TestRegister_Success(t *testing.T) {
	svc := &fakeAuthService{
		registerFn: func(_ context.Context, name, email, _ string) (*model.User, error) {
			return &model.User{ID: 1, Name: name, Email: email, PasswordHash: "secret-hash", Role: model.RoleUser}, nil
		},
	}

	w := postJSON(authRouter(svc), "/register", `{"name":"Alice","email":"alice@example.com","password":"supersecret"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body: %s)", w.Code, http.StatusCreated, w.Body.String())
	}
	body := decodeBody(t, w)
	if body["email"] != "alice@example.com" {
		t.Errorf("email = %v, want alice@example.com", body["email"])
	}
	if body["role"] != "user" {
		t.Errorf("role = %v, want user", body["role"])
	}
	if _, present := body["password_hash"]; present {
		t.Error("response must not contain password_hash")
	}
	if strings.Contains(w.Body.String(), "secret-hash") {
		t.Error("response leaks the password hash value")
	}
}

func TestRegister_InvalidInput(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"malformed JSON", `{"name":`},
		{"missing name", `{"email":"alice@example.com","password":"supersecret"}`},
		{"missing email", `{"name":"Alice","password":"supersecret"}`},
		{"invalid email", `{"name":"Alice","email":"not-an-email","password":"supersecret"}`},
		{"missing password", `{"name":"Alice","email":"alice@example.com"}`},
		{"password too short", `{"name":"Alice","email":"alice@example.com","password":"1234567"}`},
		{"password too long for bcrypt", fmt.Sprintf(`{"name":"Alice","email":"alice@example.com","password":"%s"}`, strings.Repeat("a", 73))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAuthService{}

			w := postJSON(authRouter(svc), "/register", tt.body)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
			if svc.calls != 0 {
				t.Error("the service must not be called for invalid input")
			}
		})
	}
}

func TestRegister_PasswordLengthBoundaries(t *testing.T) {
	svc := &fakeAuthService{
		registerFn: func(_ context.Context, name, email, _ string) (*model.User, error) {
			return &model.User{ID: 1, Name: name, Email: email, Role: model.RoleUser}, nil
		},
	}

	for _, length := range []int{8, 72} {
		t.Run(fmt.Sprintf("%d characters", length), func(t *testing.T) {
			body := fmt.Sprintf(`{"name":"Alice","email":"alice@example.com","password":"%s"}`, strings.Repeat("a", length))

			w := postJSON(authRouter(svc), "/register", body)

			if w.Code != http.StatusCreated {
				t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
			}
		})
	}
}

func TestRegister_ErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"email already taken", util.ErrEmailTaken, http.StatusConflict},
		{"wrapped email already taken", fmt.Errorf("create: %w", util.ErrEmailTaken), http.StatusConflict},
		{"unexpected failure", errors.New("pq: connection refused"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAuthService{
				registerFn: func(context.Context, string, string, string) (*model.User, error) {
					return nil, tt.err
				},
			}

			w := postJSON(authRouter(svc), "/register", `{"name":"Alice","email":"alice@example.com","password":"supersecret"}`)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusInternalServerError {
				if strings.Contains(w.Body.String(), "connection refused") {
					t.Error("a 500 response must not leak the underlying error")
				}
			}
		})
	}
}

func TestLogin_Success(t *testing.T) {
	svc := &fakeAuthService{
		loginFn: func(_ context.Context, email, password string) (string, error) {
			if email != "alice@example.com" || password != "supersecret" {
				t.Errorf("service received (%q, %q), want the request's credentials", email, password)
			}
			return "signed.jwt.token", nil
		},
	}

	w := postJSON(authRouter(svc), "/login", `{"email":"alice@example.com","password":"supersecret"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if token := decodeBody(t, w)["token"]; token != "signed.jwt.token" {
		t.Errorf("token = %v, want signed.jwt.token", token)
	}
}

func TestLogin_InvalidInput(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"malformed JSON", `{"email":`},
		{"missing email", `{"password":"supersecret"}`},
		{"invalid email", `{"email":"nope","password":"supersecret"}`},
		{"missing password", `{"email":"alice@example.com"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAuthService{}

			w := postJSON(authRouter(svc), "/login", tt.body)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
			if svc.calls != 0 {
				t.Error("the service must not be called for invalid input")
			}
		})
	}
}

func TestLogin_ErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"invalid credentials", util.ErrInvalidCredentials, http.StatusUnauthorized},
		{"unexpected failure", errors.New("pq: connection refused"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAuthService{
				loginFn: func(context.Context, string, string) (string, error) {
					return "", tt.err
				},
			}

			w := postJSON(authRouter(svc), "/login", `{"email":"alice@example.com","password":"supersecret"}`)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if _, present := decodeBody(t, w)["token"]; present {
				t.Error("no token may be returned on failure")
			}
			if tt.wantStatus == http.StatusInternalServerError && strings.Contains(w.Body.String(), "connection refused") {
				t.Error("a 500 response must not leak the underlying error")
			}
		})
	}
}
