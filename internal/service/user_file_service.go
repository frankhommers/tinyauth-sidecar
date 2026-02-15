package service

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"tinyauth-usermanagement/internal/config"
)

type UserRecord struct {
	Username   string
	Password   string
	TotpSecret string
}

type UserFileService struct {
	cfg *config.Config
	mu  sync.Mutex
}

func NewUserFileService(cfg *config.Config) *UserFileService {
	return &UserFileService{cfg: cfg}
}

func ParseUserLine(line string) (UserRecord, error) {
	parts := strings.SplitN(strings.TrimSpace(line), ":", 4)
	if len(parts) < 2 || len(parts) > 3 {
		return UserRecord{}, errors.New("invalid user format")
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" {
			return UserRecord{}, errors.New("invalid user format")
		}
	}
	u := UserRecord{Username: parts[0], Password: parts[1]}
	if len(parts) == 3 {
		u.TotpSecret = parts[2]
	}
	return u, nil
}

func (s *UserFileService) ReadAll() ([]UserRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readAllNoLock()
}

func (s *UserFileService) readAllNoLock() ([]UserRecord, error) {
	f, err := os.Open(s.cfg.UsersFilePath)
	if errors.Is(err, os.ErrNotExist) {
		return []UserRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var users []UserRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		u, err := ParseUserLine(line)
		if err != nil {
			return nil, fmt.Errorf("bad user line %q: %w", line, err)
		}
		users = append(users, u)
	}
	return users, scanner.Err()
}

func (s *UserFileService) Find(username string) (UserRecord, bool, error) {
	users, err := s.ReadAll()
	if err != nil {
		return UserRecord{}, false, err
	}
	for _, u := range users {
		if strings.EqualFold(u.Username, username) {
			return u, true, nil
		}
	}
	return UserRecord{}, false, nil
}

func (s *UserFileService) Upsert(user UserRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	users, err := s.readAllNoLock()
	if err != nil {
		return err
	}
	replaced := false
	for i := range users {
		if strings.EqualFold(users[i].Username, user.Username) {
			users[i] = user
			replaced = true
			break
		}
	}
	if !replaced {
		users = append(users, user)
	}
	return s.writeAllNoLock(users)
}

func (s *UserFileService) Delete(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	users, err := s.readAllNoLock()
	if err != nil {
		return err
	}
	newUsers := make([]UserRecord, 0, len(users))
	for _, u := range users {
		if !strings.EqualFold(u.Username, username) {
			newUsers = append(newUsers, u)
		}
	}
	return s.writeAllNoLock(newUsers)
}

func (s *UserFileService) writeAllNoLock(users []UserRecord) error {
	if err := os.MkdirAll(filepath.Dir(s.cfg.UsersFilePath), 0o755); err != nil {
		return err
	}
	tmp := s.cfg.UsersFilePath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	for _, u := range users {
		line := u.Username + ":" + u.Password
		if strings.TrimSpace(u.TotpSecret) != "" {
			line += ":" + strings.TrimSpace(u.TotpSecret)
		}
		if _, err := f.WriteString(line + "\n"); err != nil {
			f.Close()
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, s.cfg.UsersFilePath)
}
