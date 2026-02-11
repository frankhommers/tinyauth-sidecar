package service

import (
	"database/sql"
	"errors"
	"fmt"
	"image/png"
	"strings"
	"time"

	"tinyauth-usermanagement/internal/config"
	"tinyauth-usermanagement/internal/store"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

type AccountService struct {
	cfg    config.Config
	store  *store.SQLiteStore
	users  *UserFileService
	mail   *MailService
	docker *DockerService
}

func NewAccountService(cfg config.Config, st *store.SQLiteStore, users *UserFileService, mail *MailService, docker *DockerService) *AccountService {
	return &AccountService{cfg: cfg, store: st, users: users, mail: mail, docker: docker}
}

func (s *AccountService) RequestPasswordReset(username string) error {
	u, ok, err := s.users.Find(username)
	if err != nil {
		return err
	}
	if !ok {
		return nil // don't leak
	}
	token := uuid.NewString()
	exp := time.Now().Add(time.Duration(s.cfg.ResetTokenTTLSeconds) * time.Second).Unix()
	_, err = s.store.DB.Exec(`INSERT INTO reset_tokens(token, username, expires_at, used) VALUES(?,?,?,0)`, token, u.Username, exp)
	if err != nil {
		return err
	}
	return s.mail.SendResetEmail(u.Username, token)
}

func (s *AccountService) ResetPassword(token, newPassword string) error {
	var username string
	var expires int64
	var used int
	err := s.store.DB.QueryRow(`SELECT username, expires_at, used FROM reset_tokens WHERE token = ?`, token).Scan(&username, &expires, &used)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("invalid token")
		}
		return err
	}
	if used == 1 || time.Now().Unix() > expires {
		return errors.New("token expired")
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	u, ok, err := s.users.Find(username)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("user not found")
	}
	u.Password = hash
	if err := s.users.Upsert(u); err != nil {
		return err
	}
	_, _ = s.store.DB.Exec(`UPDATE reset_tokens SET used = 1 WHERE token = ?`, token)
	s.docker.RestartTinyauth()
	return nil
}

func (s *AccountService) Signup(username, email, password string) (string, error) {
	if username == "" || password == "" {
		return "", errors.New("username and password required")
	}
	if existing, ok, err := s.users.Find(username); err != nil {
		return "", err
	} else if ok && existing.Username != "" {
		return "", errors.New("user already exists")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return "", err
	}
	if s.cfg.SignupRequireApproval {
		id := uuid.NewString()
		_, err = s.store.DB.Exec(`INSERT INTO pending_signups(id, username, email, password_hash, created_at, approved) VALUES(?,?,?,?,?,0)`, id, username, email, hash, time.Now().Unix())
		if err != nil {
			return "", err
		}
		return id, nil
	}
	if err := s.users.Upsert(UserRecord{Username: username, Password: hash}); err != nil {
		return "", err
	}
	s.docker.RestartTinyauth()
	return "approved", nil
}

func (s *AccountService) ApproveSignup(id string) error {
	var username, hash string
	if err := s.store.DB.QueryRow(`SELECT username, password_hash FROM pending_signups WHERE id = ?`, id).Scan(&username, &hash); err != nil {
		return err
	}
	if err := s.users.Upsert(UserRecord{Username: username, Password: hash}); err != nil {
		return err
	}
	_, _ = s.store.DB.Exec(`UPDATE pending_signups SET approved = 1 WHERE id = ?`, id)
	s.docker.RestartTinyauth()
	return nil
}

func (s *AccountService) Profile(username string) (map[string]any, error) {
	u, ok, err := s.users.Find(username)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("not found")
	}
	return map[string]any{"username": u.Username, "totpEnabled": strings.TrimSpace(u.TotpSecret) != ""}, nil
}

func (s *AccountService) ChangePassword(username, oldPassword, newPassword string) error {
	u, ok, err := s.users.Find(username)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("not found")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(oldPassword)) != nil {
		return errors.New("old password invalid")
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	u.Password = hash
	if err := s.users.Upsert(u); err != nil {
		return err
	}
	s.docker.RestartTinyauth()
	return nil
}

func (s *AccountService) TotpSetup(username string) (secret, otpURL string, pngBytes []byte, err error) {
	key, err := totp.Generate(totp.GenerateOpts{Issuer: s.cfg.TOTPIssuer, AccountName: username})
	if err != nil {
		return "", "", nil, err
	}
	img, err := key.Image(256, 256)
	if err != nil {
		return "", "", nil, err
	}
	b := new(bytesBuffer)
	if err := png.Encode(b, img); err != nil {
		return "", "", nil, err
	}
	return key.Secret(), key.URL(), b.Bytes(), nil
}

type bytesBuffer struct{ b []byte }

func (w *bytesBuffer) Write(p []byte) (int, error) { w.b = append(w.b, p...); return len(p), nil }
func (w *bytesBuffer) Bytes() []byte               { return w.b }

func (s *AccountService) TotpEnable(username, secret, code string) error {
	if !totp.Validate(code, secret) {
		return errors.New("invalid code")
	}
	u, ok, err := s.users.Find(username)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("not found")
	}
	u.TotpSecret = secret
	if err := s.users.Upsert(u); err != nil {
		return err
	}
	s.docker.RestartTinyauth()
	return nil
}

func (s *AccountService) TotpDisable(username, password string) error {
	u, ok, err := s.users.Find(username)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("not found")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		return errors.New("invalid password")
	}
	u.TotpSecret = ""
	if err := s.users.Upsert(u); err != nil {
		return err
	}
	s.docker.RestartTinyauth()
	return nil
}

func (s *AccountService) TotpRecover(username, recoveryKey, newSecret, code string) error {
	if recoveryKey != fmt.Sprintf("RECOVERY-%s", username) {
		return errors.New("invalid recovery key")
	}
	return s.TotpEnable(username, newSecret, code)
}

func (s *AccountService) ValidateToken(token string) (*otp.Key, error) {
	return otp.NewKeyFromURL(token)
}
