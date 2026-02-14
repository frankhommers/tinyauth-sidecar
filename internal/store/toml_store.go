package store

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

// UserMeta holds persistent per-user metadata stored in users.toml.
type UserMeta struct {
	Name     string `toml:"name,omitempty"`
	Role     string `toml:"role,omitempty"`
	Phone    string `toml:"phone,omitempty"`
	Approved bool   `toml:"approved,omitempty"`
}

// resetTokenEntry is an in-memory reset token record.
type resetTokenEntry struct {
	Username  string
	ExpiresAt int64
	Used      bool
}

// pendingSignup is an in-memory pending signup record.
type pendingSignup struct {
	Username     string
	Email        string
	PasswordHash string
	CreatedAt    int64
	Approved     bool
}

// smsResetCode is an in-memory SMS reset code record.
type smsResetCode struct {
	Username  string
	Code      string
	ExpiresAt int64
	Used      bool
}

// Store provides persistence via a TOML file for user metadata and
// in-memory maps for ephemeral data (sessions, tokens, signups, SMS codes).
type Store struct {
	tomlPath string

	mu    sync.RWMutex
	users map[string]*UserMeta // key = email/username

	resetMu     sync.Mutex
	resetTokens map[string]*resetTokenEntry // key = token

	signupMu sync.Mutex
	signups  map[string]*pendingSignup // key = id

	smsMu    sync.Mutex
	smsCodes map[string]*smsResetCode // key = id
}

// NewStore creates a new TOML-backed store. It reads the TOML file
// (or creates an empty one) and initializes in-memory maps.
func NewStore(tomlPath string) (*Store, error) {
	if tomlPath == "" {
		tomlPath = os.Getenv("USERS_TOML")
		if tomlPath == "" {
			tomlPath = "/users/users.toml"
		}
	}

	if err := os.MkdirAll(filepath.Dir(tomlPath), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir toml dir: %w", err)
	}

	s := &Store{
		tomlPath:    tomlPath,
		users:       make(map[string]*UserMeta),
		resetTokens: make(map[string]*resetTokenEntry),
		signups:     make(map[string]*pendingSignup),
		smsCodes:    make(map[string]*smsResetCode),
	}

	// Load existing TOML file if present
	if _, err := os.Stat(tomlPath); err == nil {
		if _, err := toml.DecodeFile(tomlPath, &s.users); err != nil {
			return nil, fmt.Errorf("decode users.toml: %w", err)
		}
	}

	return s, nil
}

// Close is a no-op for the TOML store (satisfies the old interface).
func (s *Store) Close() error { return nil }

// ---------- TOML persistence helpers ----------

func (s *Store) saveTOML() error {
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(s.users); err != nil {
		return fmt.Errorf("encode users.toml: %w", err)
	}

	// Atomic write: temp file + rename
	tmp := s.tomlPath + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write temp toml: %w", err)
	}
	if err := os.Rename(tmp, s.tomlPath); err != nil {
		return fmt.Errorf("rename toml: %w", err)
	}
	return nil
}

// ---------- User metadata (TOML-persisted) ----------

// SetPhone sets the phone number for a user.
func (s *Store) SetPhone(username, phone string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	meta, ok := s.users[username]
	if !ok {
		meta = &UserMeta{}
		s.users[username] = meta
	}
	meta.Phone = phone
	return s.saveTOML()
}

// GetPhone retrieves the phone number for a user.
func (s *Store) GetPhone(username string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	meta, ok := s.users[username]
	if !ok {
		return "", nil
	}
	return meta.Phone, nil
}

// FindUserByPhone returns the username for a given phone number.
func (s *Store) FindUserByPhone(phone string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for username, meta := range s.users {
		if meta.Phone == phone {
			return username, nil
		}
	}
	return "", nil
}

// GetUserMeta returns the metadata for a user (or nil if not found).
func (s *Store) GetUserMeta(username string) *UserMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if meta, ok := s.users[username]; ok {
		cp := *meta
		return &cp
	}
	return nil
}

// SetUserMeta sets/replaces the metadata for a user.
func (s *Store) SetUserMeta(username string, meta *UserMeta) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.users[username] = meta
	return s.saveTOML()
}

// ---------- Reset tokens (in-memory) ----------

// CreateResetToken stores a new password reset token.
func (s *Store) CreateResetToken(token, username string, expiresAt int64) error {
	s.resetMu.Lock()
	defer s.resetMu.Unlock()

	s.resetTokens[token] = &resetTokenEntry{
		Username:  username,
		ExpiresAt: expiresAt,
		Used:      false,
	}
	return nil
}

// GetResetToken retrieves a reset token. Returns username, expiresAt, used.
// Returns empty username if not found.
func (s *Store) GetResetToken(token string) (username string, expiresAt int64, used bool, err error) {
	s.resetMu.Lock()
	defer s.resetMu.Unlock()

	rt, ok := s.resetTokens[token]
	if !ok {
		return "", 0, false, nil
	}
	return rt.Username, rt.ExpiresAt, rt.Used, nil
}

// MarkResetTokenUsed marks a reset token as used.
func (s *Store) MarkResetTokenUsed(token string) error {
	s.resetMu.Lock()
	defer s.resetMu.Unlock()

	if rt, ok := s.resetTokens[token]; ok {
		rt.Used = true
	}
	return nil
}

// ---------- Pending signups (in-memory) ----------

// CreatePendingSignup stores a new pending signup.
func (s *Store) CreatePendingSignup(id, username, email, passwordHash string, createdAt int64) error {
	s.signupMu.Lock()
	defer s.signupMu.Unlock()

	s.signups[id] = &pendingSignup{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    createdAt,
		Approved:     false,
	}
	return nil
}

// GetPendingSignup retrieves a pending signup by id.
// Returns username and passwordHash. Returns empty username if not found.
func (s *Store) GetPendingSignup(id string) (username, passwordHash string, err error) {
	s.signupMu.Lock()
	defer s.signupMu.Unlock()

	ps, ok := s.signups[id]
	if !ok {
		return "", "", fmt.Errorf("signup not found")
	}
	return ps.Username, ps.PasswordHash, nil
}

// ApprovePendingSignup marks a pending signup as approved.
func (s *Store) ApprovePendingSignup(id string) error {
	s.signupMu.Lock()
	defer s.signupMu.Unlock()

	if ps, ok := s.signups[id]; ok {
		ps.Approved = true
	}
	return nil
}

// ---------- SMS reset codes (in-memory) ----------

// StoreSMSResetCode stores a reset code for SMS-based password reset.
func (s *Store) StoreSMSResetCode(id, username, code string, expiresAt int64) error {
	s.smsMu.Lock()
	defer s.smsMu.Unlock()

	s.smsCodes[id] = &smsResetCode{
		Username:  username,
		Code:      code,
		ExpiresAt: expiresAt,
		Used:      false,
	}
	return nil
}

// VerifySMSResetCode checks if a code is valid for the given phone's user.
func (s *Store) VerifySMSResetCode(phone, code string) (string, error) {
	username, err := s.FindUserByPhone(phone)
	if err != nil {
		return "", err
	}
	if username == "" {
		return "", fmt.Errorf("no user with that phone")
	}

	s.smsMu.Lock()
	defer s.smsMu.Unlock()

	// Find the most recent matching code for this user
	var bestID string
	var bestExpires int64
	for id, sc := range s.smsCodes {
		if sc.Username == username && sc.Code == code && sc.ExpiresAt > bestExpires {
			bestID = id
			bestExpires = sc.ExpiresAt
		}
	}

	if bestID == "" {
		return "", fmt.Errorf("invalid code")
	}

	sc := s.smsCodes[bestID]
	if sc.Used {
		return "", fmt.Errorf("code already used")
	}
	if time.Now().Unix() > sc.ExpiresAt {
		return "", fmt.Errorf("code expired")
	}

	sc.Used = true
	return username, nil
}
