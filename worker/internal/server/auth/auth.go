package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/store"
)

const SessionLifetime = 7 * 24 * time.Hour

type AdminUser struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    *time.Time
	LastLogin    *time.Time
}

type Admin struct {
	User   *AdminUser
	Master bool
}

type Session struct {
	Token     string
	ExpiresAt time.Time
}

type Service struct {
	db        *store.Store
	masterKey string
}

func New(db *store.Store, masterKey string) *Service {
	return &Service{db: db, masterKey: masterKey}
}

func (user *AdminUser) ToMap() map[string]any {
	return map[string]any{
		"id":         user.ID,
		"username":   user.Username,
		"created_at": pytime.IsoformatOrNil(user.CreatedAt),
		"last_login": pytime.IsoformatOrNil(user.LastLogin),
	}
}

func ExtractToken(r *http.Request) string {
	authorization := r.Header.Get("Authorization")
	if strings.HasPrefix(authorization, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
	}
	return strings.TrimSpace(r.Header.Get("X-API-Key"))
}

func (service *Service) IsMasterKey(token string) bool {
	return token != "" && subtle.ConstantTimeCompare([]byte(token), []byte(service.masterKey)) == 1
}

func (service *Service) RequireAPIKey(r *http.Request) error {
	if r.Header.Get("X-API-Key") != service.masterKey {
		return httpx.Unauthorized("Invalid API key")
	}
	return nil
}

func (service *Service) CurrentAdmin(ctx context.Context, r *http.Request) (*Admin, error) {
	token := ExtractToken(r)
	user, err := service.sessionUser(ctx, token)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return &Admin{User: user}, nil
	}
	if service.IsMasterKey(token) {
		return &Admin{Master: true}, nil
	}
	return nil, nil
}

func (service *Service) RequireAdmin(r *http.Request) (*Admin, error) {
	admin, err := service.CurrentAdmin(r.Context(), r)
	if err != nil {
		return nil, err
	}
	if admin == nil {
		return nil, httpx.Unauthorized("Admin authentication required")
	}
	return admin, nil
}

func (service *Service) sessionUser(ctx context.Context, token string) (*AdminUser, error) {
	if token == "" {
		return nil, nil
	}
	var userID string
	var expiresAt *time.Time
	err := service.db.Pool.QueryRow(ctx,
		"SELECT user_id, expires_at FROM admin_sessions WHERE token = $1", token).Scan(&userID, &expiresAt)
	if store.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	now := pytime.Now()
	if expiresAt != nil && expiresAt.Before(now) {
		_, err := service.db.Pool.Exec(ctx, "DELETE FROM admin_sessions WHERE token = $1", token)
		return nil, err
	}
	if _, err := service.db.Pool.Exec(ctx, "UPDATE admin_sessions SET last_seen = $2 WHERE token = $1", token, now); err != nil {
		return nil, err
	}
	return service.UserByID(ctx, userID)
}

func (service *Service) UserByID(ctx context.Context, userID string) (*AdminUser, error) {
	return service.findUser(ctx, "id", userID)
}

func (service *Service) UserByUsername(ctx context.Context, username string) (*AdminUser, error) {
	return service.findUser(ctx, "username", username)
}

func (service *Service) findUser(ctx context.Context, column string, value string) (*AdminUser, error) {
	user := &AdminUser{}
	err := service.db.Pool.QueryRow(ctx,
		"SELECT id, username, password_hash, created_at, last_login FROM admin_users WHERE "+column+" = $1", value).
		Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.LastLogin)
	if store.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (service *Service) CreateSession(ctx context.Context, user *AdminUser) (Session, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return Session{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	now := pytime.Now()
	session := Session{Token: token, ExpiresAt: now.Add(SessionLifetime)}
	_, err := service.db.Pool.Exec(ctx,
		"INSERT INTO admin_sessions (token, user_id, created_at, expires_at, last_seen) VALUES ($1, $2, $3, $4, $3)",
		token, user.ID, now, session.ExpiresAt)
	if err != nil {
		return Session{}, err
	}
	_, err = service.db.Pool.Exec(ctx, "UPDATE admin_users SET last_login = $2 WHERE id = $1", user.ID, now)
	return session, err
}

func (service *Service) RevokeToken(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	_, err := service.db.Pool.Exec(ctx, "DELETE FROM admin_sessions WHERE token = $1", token)
	return err
}

func (service *Service) CreateUser(ctx context.Context, username string, password string) (*AdminUser, error) {
	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	now := pytime.Now()
	user := &AdminUser{ID: uuid.NewString(), Username: username, PasswordHash: passwordHash, CreatedAt: &now}
	_, err = service.db.Pool.Exec(ctx,
		"INSERT INTO admin_users (id, username, password_hash, created_at) VALUES ($1, $2, $3, $4)",
		user.ID, user.Username, user.PasswordHash, now)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (service *Service) SetPassword(ctx context.Context, userID string, password string) error {
	passwordHash, err := HashPassword(password)
	if err != nil {
		return err
	}
	_, err = service.db.Pool.Exec(ctx, "UPDATE admin_users SET password_hash = $2 WHERE id = $1", userID, passwordHash)
	return err
}
