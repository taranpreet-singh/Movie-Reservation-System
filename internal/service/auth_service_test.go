package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"movie-reservation-system/internal/model"
	"movie-reservation-system/internal/util"
)

const testSecret = "test-secret"

type fakeUserRepo struct {
	byEmail   map[string]*model.User
	createErr error
	getErr    error
	nextID    int64
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byEmail: map[string]*model.User{}, nextID: 1}
}

func (f *fakeUserRepo) Create(_ context.Context, u *model.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	u.ID = f.nextID
	f.nextID++
	u.CreatedAt = time.Now()
	u.UpdatedAt = u.CreatedAt
	f.byEmail[u.Email] = u
	return nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	u, ok := f.byEmail[email]
	if !ok {
		return nil, util.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) GetByID(_ context.Context, id int64) (*model.User, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	for _, u := range f.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, util.ErrNotFound
}

func TestRegister_CreatesUserWithSafeDefaults(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewAuthService(repo, testSecret)

	user, err := svc.Register(context.Background(), "  Alice  ", "  Alice@Example.COM ", "supersecret")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if user.Role != model.RoleUser {
		t.Errorf("Role = %q, want %q", user.Role, model.RoleUser)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("Email = %q, want normalized %q", user.Email, "alice@example.com")
	}
	if user.Name != "Alice" {
		t.Errorf("Name = %q, want trimmed %q", user.Name, "Alice")
	}
	if user.PasswordHash == "" || user.PasswordHash == "supersecret" {
		t.Error("password must be stored as a hash, not plaintext")
	}
	if err := util.CheckPassword(user.PasswordHash, "supersecret"); err != nil {
		t.Errorf("stored hash does not verify against the original password: %v", err)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := newFakeUserRepo()
	repo.createErr = util.ErrEmailTaken
	svc := NewAuthService(repo, testSecret)

	_, err := svc.Register(context.Background(), "Alice", "alice@example.com", "supersecret")
	if !errors.Is(err, util.ErrEmailTaken) {
		t.Fatalf("err = %v, want util.ErrEmailTaken", err)
	}
}

func TestRegister_RepositoryFailure(t *testing.T) {
	repo := newFakeUserRepo()
	boom := errors.New("db down")
	repo.createErr = boom
	svc := NewAuthService(repo, testSecret)

	_, err := svc.Register(context.Background(), "Alice", "alice@example.com", "supersecret")
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the repository error to pass through", err)
	}
}

func TestLogin_Success(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewAuthService(repo, testSecret)

	registered, err := svc.Register(context.Background(), "Alice", "alice@example.com", "supersecret")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	token, err := svc.Login(context.Background(), "  ALICE@example.com ", "supersecret")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	claims, err := util.ParseToken(token, testSecret)
	if err != nil {
		t.Fatalf("login token does not parse: %v", err)
	}
	if claims.UserID != registered.ID {
		t.Errorf("token UserID = %d, want %d", claims.UserID, registered.ID)
	}
	if claims.Role != string(model.RoleUser) {
		t.Errorf("token Role = %q, want %q", claims.Role, model.RoleUser)
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewAuthService(repo, testSecret)
	if _, err := svc.Register(context.Background(), "Alice", "alice@example.com", "supersecret"); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{"unknown email", "nobody@example.com", "supersecret"},
		{"wrong password", "alice@example.com", "not-the-password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := svc.Login(context.Background(), tt.email, tt.password)
			if !errors.Is(err, util.ErrInvalidCredentials) {
				t.Fatalf("err = %v, want util.ErrInvalidCredentials", err)
			}
			if token != "" {
				t.Error("no token may be returned on failure")
			}
		})
	}
}

func TestLogin_RepositoryFailureIsNotMaskedAsBadCredentials(t *testing.T) {
	repo := newFakeUserRepo()
	boom := errors.New("db down")
	repo.getErr = boom
	svc := NewAuthService(repo, testSecret)

	_, err := svc.Login(context.Background(), "alice@example.com", "supersecret")
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the repository error", err)
	}
	if errors.Is(err, util.ErrInvalidCredentials) {
		t.Fatal("an infrastructure failure must not be reported as invalid credentials")
	}
}
