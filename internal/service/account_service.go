package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"image/png"
	"log"
	"math/big"
	"regexp"
	"strings"
	"time"

	"tinyauth-sidecar/internal/config"
	"tinyauth-sidecar/internal/provider"
	"tinyauth-sidecar/internal/store"

	"github.com/google/uuid"
	zxcvbn "github.com/nbutton23/zxcvbn-go"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type AccountService struct {
	cfg             *config.Config
	store           *store.Store
	users           *UserFileService
	mail            *MailService
	docker          *DockerService
	passwordTargets *provider.PasswordTargetProvider
	passwordHooks   []provider.PasswordChangeHook
	sms             provider.SMSProvider
	audit           *AuditService
}

func NewAccountService(cfg *config.Config, st *store.Store, users *UserFileService, mail *MailService, docker *DockerService, passwordTargets *provider.PasswordTargetProvider, sms provider.SMSProvider, audit *AuditService, passwordHooks ...provider.PasswordChangeHook) *AccountService {
	var hooks []provider.PasswordChangeHook
	for _, h := range passwordHooks {
		if h != nil {
			hooks = append(hooks, h)
		}
	}
	return &AccountService{cfg: cfg, store: st, users: users, mail: mail, docker: docker, passwordTargets: passwordTargets, passwordHooks: hooks, sms: sms, audit: audit}
}

func (s *AccountService) RequestPasswordReset(username, clientIP string) error {
	u, ok, err := s.users.Find(username)
	if err != nil {
		return err
	}

	// If username_is_email is false, also try finding user by email
	if !ok && !s.cfg.UsernameIsEmail {
		foundUser, findErr := s.store.FindUserByEmail(username)
		if findErr != nil {
			return findErr
		}
		if foundUser != "" {
			u, ok, err = s.users.Find(foundUser)
			if err != nil {
				return err
			}
		}
	}

	if !ok {
		return nil // don't leak
	}

	token := uuid.NewString()
	exp := time.Now().Add(time.Duration(s.cfg.ResetTokenTTLSeconds) * time.Second).Unix()
	if err := s.store.CreateResetToken(token, u.Username, exp); err != nil {
		return err
	}

	// Determine email recipient
	toEmail := u.Username // default: username is the email
	if !s.cfg.UsernameIsEmail {
		email, _ := s.store.GetEmail(u.Username)
		if email == "" {
			log.Printf("[reset] user %s has no email address configured", u.Username)
			return errors.New("no email address configured")
		}
		toEmail = email
	}

	s.audit.Log("password_reset_request", username, clientIP, "sent")
	return s.mail.SendResetEmail(toEmail, token)
}

func (s *AccountService) ResetPassword(token, newPassword, clientIP string) error {
	username, expiresAt, used, err := s.store.GetResetToken(token)
	if err != nil {
		return err
	}
	if username == "" {
		s.audit.Log("password_reset_confirm", "unknown", clientIP, "invalid_token")
		return errors.New("invalid token")
	}
	if used || time.Now().Unix() > expiresAt {
		s.audit.Log("password_reset_confirm", username, clientIP, "token_expired")
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
	_ = s.store.MarkResetTokenUsed(token)
	s.docker.RestartTinyauth()
	s.syncPasswordTargets(username, newPassword, hash)
	s.notifyPasswordChanged(username)
	s.audit.Log("password_reset_confirm", username, clientIP, "success")
	return nil
}

func (s *AccountService) Signup(username, email, password string) (string, error) {
	return s.SignupWithPhone(username, email, password, "")
}

func (s *AccountService) validatePassword(password string) error {
	if len(password) < s.cfg.MinPasswordLength {
		return fmt.Errorf("password_too_short")
	}
	result := zxcvbn.PasswordStrength(password, nil)
	if result.Score < s.cfg.MinPasswordStrength {
		return fmt.Errorf("password_too_weak")
	}
	return nil
}

func (s *AccountService) SignupWithPhone(username, email, password, phone string) (string, error) {
	if username == "" || password == "" {
		return "", errors.New("username and password required")
	}
	if s.cfg.UsernameIsEmail {
		if !emailRegex.MatchString(username) {
			return "", errors.New("username must be a valid email address")
		}
	}
	if err := s.validatePassword(password); err != nil {
		return "", err
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
		if err := s.store.CreatePendingSignup(id, username, email, hash, time.Now().Unix()); err != nil {
			return "", err
		}
		if phone != "" {
			_ = s.store.SetPhone(username, phone)
		}
		if !s.cfg.UsernameIsEmail && email != "" {
			_ = s.store.SetEmail(username, email)
		}
		return id, nil
	}
	if err := s.users.Upsert(UserRecord{Username: username, Password: hash}); err != nil {
		return "", err
	}
	if phone != "" {
		_ = s.store.SetPhone(username, phone)
	}
	if !s.cfg.UsernameIsEmail && email != "" {
		_ = s.store.SetEmail(username, email)
	}
	s.docker.RestartTinyauth()
	s.syncPasswordTargets(username, password, hash)
	return "approved", nil
}

func (s *AccountService) ApproveSignup(id string) error {
	username, hash, err := s.store.GetPendingSignup(id)
	if err != nil {
		return err
	}
	if err := s.users.Upsert(UserRecord{Username: username, Password: hash}); err != nil {
		return err
	}
	_ = s.store.ApprovePendingSignup(id)
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
	phone, _ := s.store.GetPhone(username)
	email, _ := s.store.GetEmail(username)
	role := ""
	if meta := s.store.GetUserMeta(username); meta != nil {
		role = meta.Role
	}
	return map[string]any{
		"username":    u.Username,
		"totpEnabled": strings.TrimSpace(u.TotpSecret) != "",
		"phone":       phone,
		"email":       email,
		"role":        role,
	}, nil
}

func (s *AccountService) SetPhone(username, phone string) error {
	return s.store.SetPhone(username, phone)
}

func (s *AccountService) SetEmail(username, email string) error {
	return s.store.SetEmail(username, email)
}

func (s *AccountService) ChangePassword(username, oldPassword, newPassword, clientIP string) error {
	u, ok, err := s.users.Find(username)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("not found")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(oldPassword)) != nil {
		s.audit.Log("password_change", username, clientIP, "invalid_old_password")
		return errors.New("old password invalid")
	}
	if err := s.validatePassword(newPassword); err != nil {
		return err
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
	s.syncPasswordTargets(username, newPassword, hash)
	s.notifyPasswordChanged(username)
	s.audit.Log("password_change", username, clientIP, "success")
	return nil
}

// syncPasswordTargets sends password to all configured webhook targets (fire and forget).
func (s *AccountService) syncPasswordTargets(username, plainPassword, hashedPassword string) {
	if s.passwordTargets != nil {
		go func() {
			errs := s.passwordTargets.SyncPassword(username, plainPassword, hashedPassword)
			for _, err := range errs {
				log.Printf("[password-targets] sync error: %v", err)
			}
		}()
	}

	// Look up role for hook filters
	role := ""
	if meta := s.store.GetUserMeta(username); meta != nil {
		role = meta.Role
	}

	for _, hook := range s.passwordHooks {
		go func(h provider.PasswordChangeHook) {
			if err := h.OnPasswordChanged(provider.PasswordChangeContext{
				Email:    username,
				Password: plainPassword,
				Role:     role,
			}); err != nil {
				log.Printf("[password-hook] warning: %v", err)
			}
		}(hook)
	}
}

// RequestSMSReset sends a reset code via SMS.
func (s *AccountService) RequestSMSReset(phone, clientIP string) error {
	if s.sms == nil {
		return errors.New("SMS not configured")
	}
	username, err := s.store.FindUserByPhone(phone)
	if err != nil {
		return err
	}
	if username == "" {
		// Don't leak whether phone exists
		return nil
	}

	// Cooldown: max 1 SMS per phone per 5 minutes
	if s.store.HasRecentSMSCode(username, 5*time.Minute) {
		return nil // silent, don't leak info
	}

	code, err := generateNumericCode(6)
	if err != nil {
		return err
	}

	id := uuid.NewString()
	expiresAt := time.Now().Add(10 * time.Minute).Unix()
	if err := s.store.StoreSMSResetCode(id, username, code, expiresAt); err != nil {
		return err
	}

	msg := fmt.Sprintf("Your password reset code is: %s (valid for 10 minutes)", code)
	if err := s.sms.SendSMS(phone, msg); err != nil {
		log.Printf("[sms] failed to send SMS to %s: %v", phone, err)
		s.audit.Log("sms_reset_request", phone, clientIP, "send_failed")
		return fmt.Errorf("failed to send SMS")
	}

	s.audit.Log("sms_reset_request", phone, clientIP, "sent")
	return nil
}

// ResetPasswordSMS verifies a code and resets the password.
func (s *AccountService) ResetPasswordSMS(phone, code, newPassword, clientIP string) error {
	username, err := s.store.VerifySMSResetCode(phone, code)
	if err != nil {
		s.audit.Log("sms_reset_confirm", phone, clientIP, "failed:"+err.Error())
		return err
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

	s.docker.RestartTinyauth()
	s.syncPasswordTargets(username, newPassword, hash)
	s.notifyPasswordChanged(username)
	s.audit.Log("sms_reset_confirm", phone, clientIP, "success")
	return nil
}

// SMSEnabled returns true if SMS provider is configured.
func (s *AccountService) SMSEnabled() bool {
	return s.sms != nil
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

// notifyPasswordChanged sends an email notification about the password change.
func (s *AccountService) notifyPasswordChanged(username string) {
	toEmail := username
	if !s.cfg.UsernameIsEmail {
		email, _ := s.store.GetEmail(username)
		if email != "" {
			toEmail = email
		}
	}
	if toEmail == "" || !emailRegex.MatchString(toEmail) {
		return
	}
	go func() {
		if err := s.mail.SendPasswordChangedEmail(toEmail); err != nil {
			log.Printf("[mail] failed to send password changed notification to %s: %v", toEmail, err)
		}
	}()
}

// generateNumericCode generates a cryptographically random numeric code of the given length.
func generateNumericCode(length int) (string, error) {
	code := make([]byte, length)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code[i] = byte('0' + n.Int64())
	}
	return string(code), nil
}
