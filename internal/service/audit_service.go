package service

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditService logs security-relevant events to a file.
type AuditService struct {
	mu   sync.Mutex
	path string
}

// NewAuditService creates an audit logger writing to the given path.
func NewAuditService(path string) *AuditService {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("[audit] failed to create dir for %s: %v", path, err)
	}
	return &AuditService{path: path}
}

// Log writes a structured audit entry.
func (a *AuditService) Log(event, identity, ip, result string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	line := fmt.Sprintf("%s | event=%s | identity=%s | ip=%s | result=%s\n",
		time.Now().Format("2006-01-02T15:04:05Z07:00"), event, identity, ip, result)

	f, err := os.OpenFile(a.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("[audit] failed to open %s: %v", a.path, err)
		return
	}
	defer f.Close()
	f.WriteString(line)
}
