package license

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"
)

type Status string

const (
	Active  Status = "ACTIVE"
	Expired Status = "EXPIRED"
	Revoked Status = "REVOKED"
	Banned  Status = "BANNED"
)

type Validator struct {
	mu           sync.RWMutex
	secret       []byte
	cacheExpiry  time.Time
	cachedStatus Status
	offlineGrace time.Duration
	lastOnline   time.Time
}

func New(secret string) *Validator {
	return &Validator{
		secret:       []byte(secret),
		offlineGrace: 24 * time.Hour,
		lastOnline:   time.Now(),
		cachedStatus: Active,
		cacheExpiry:  time.Now().Add(1 * time.Hour),
	}
}

func HWID() string {
	hostname, _ := os.Hostname()
	data := hostname + ":" + os.Getenv("USER") + ":" + os.Getenv("HOME")
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:16])
}

func (v *Validator) Sign(payload string) string {
	mac := hmac.New(sha256.New, v.secret)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func (v *Validator) Validate(remoteStatus Status, signature, payload string) (Status, error) {
	expected := v.Sign(payload)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return Revoked, fmt.Errorf("invalid license signature")
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	v.lastOnline = time.Now()
	v.cachedStatus = remoteStatus
	v.cacheExpiry = time.Now().Add(1 * time.Hour)

	if remoteStatus != Active {
		return remoteStatus, fmt.Errorf("license status: %s", remoteStatus)
	}
	return Active, nil
}

func (v *Validator) Cached() (Status, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if time.Now().Before(v.cacheExpiry) {
		return v.cachedStatus, nil
	}
	if time.Since(v.lastOnline) <= v.offlineGrace {
		return v.cachedStatus, nil
	}
	return Expired, fmt.Errorf("offline grace period exceeded")
}
