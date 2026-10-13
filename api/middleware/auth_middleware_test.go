package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"movie-reservation-system/internal/util"

	"github.com/gin-gonic/gin"
)

const testSecret = "test-secret"

func init() {
	gin.SetMode(gin.TestMode)
}

func performRequest(r http.Handler, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuthRequired_RejectsBadRequests(t *testing.T) {
	validToken, err := util.GenerateToken(1, "user", testSecret)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	otherSecretToken, err := util.GenerateToken(1, "user", "another-secret")
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	tests := []struct {
		name   string
		header string
	}{
		{"no header", ""},
		{"wrong scheme", "Basic " + validToken},
		{"scheme without token", "Bearer"},
		{"empty token", "Bearer "},
		{"token without scheme", validToken},
		{"garbage token", "Bearer not-a-real-token"},
		{"token signed with another secret", "Bearer " + otherSecretToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reached := false
			r := gin.New()
			r.GET("/protected", AuthRequired(testSecret), func(ctx *gin.Context) {
				reached = true
				ctx.Status(http.StatusOK)
			})

			w := performRequest(r, tt.header)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
			}
			if reached {
				t.Error("the protected handler must not run when authentication fails")
			}
		})
	}
}

func TestAuthRequired_AcceptsValidToken(t *testing.T) {
	token, err := util.GenerateToken(7, "admin", testSecret)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	for _, header := range []string{"Bearer " + token, "bearer " + token} {
		t.Run(header[:6], func(t *testing.T) {
			var gotID any
			var gotRole any
			r := gin.New()
			r.GET("/protected", AuthRequired(testSecret), func(ctx *gin.Context) {
				gotID, _ = ctx.Get(util.CtxUserID)
				gotRole, _ = ctx.Get(util.CtxRole)
				ctx.Status(http.StatusOK)
			})

			w := performRequest(r, header)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
			}
			if id, ok := gotID.(int64); !ok || id != 7 {
				t.Errorf("context user id = %#v, want int64(7)", gotID)
			}
			if gotRole != "admin" {
				t.Errorf("context role = %#v, want %q", gotRole, "admin")
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	setRole := func(role any) gin.HandlerFunc {
		return func(ctx *gin.Context) {
			if role != nil {
				ctx.Set(util.CtxRole, role)
			}
		}
	}

	tests := []struct {
		name       string
		role       any
		wantStatus int
	}{
		{"matching role", "admin", http.StatusOK},
		{"different role", "user", http.StatusForbidden},
		{"role missing from context", nil, http.StatusForbidden},
		{"role of the wrong type", 1, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/protected", setRole(tt.role), RequireRole("admin"), func(ctx *gin.Context) {
				ctx.Status(http.StatusOK)
			})

			w := performRequest(r, "")

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
