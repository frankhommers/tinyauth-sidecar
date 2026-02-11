package service

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"tinyauth-usermanagement/internal/config"
	"tinyauth-usermanagement/internal/store"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	cfg   config.Config
	store *store.SQLiteStore
	users *UserFileService
}

func NewAuthService(cfg config.Config, st *store.SQLiteStore, users *UserFileService) *AuthService {
	return &AuthService{cfg: cfg, store: st, users: users}
}

func (s *AuthService) Login(username, password string) (string, error) {
	u, ok, err := s.users.Find(username)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.New("invalid credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		return "", errors.New("invalid credentials")
	}

	token, err := randomToken(32)
	if err != nil {
		return "", err
	}
	now := time.Now().Unix()
	expires := now + s.cfg.SessionTTLSeconds
	_, err = s.store.DB.Exec(`INSERT INTO sessions(token, username, created_at, expires_at) VALUES(?,?,?,?)`, token, u.Username, now, expires)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *AuthService) Logout(token string) error {
	_, err := s.store.DB.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

func (s *AuthService) SessionUsername(token string) (string, error) {
	var username string
	var expires int64
	err := s.store.DB.QueryRow(`SELECT username, expires_at FROM sessions WHERE token = ?`, token).Scan(&username, &expires)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("unauthorized")
		}
		return "", err
	}
	if time.Now().Unix() > expires {
		_, _ = s.store.DB.Exec(`DELETE FROM sessions WHERE token = ?`, token)
		return "", errors.New("unauthorized")
	}
	return username, nil
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
