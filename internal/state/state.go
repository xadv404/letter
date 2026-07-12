package state

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type DomainProgress struct {
	Domain     string    `json:"domain"`
	Pages      int       `json:"pages"`
	Errors     int       `json:"errors"`
	Finished   bool      `json:"finished"`
	Visited    []string  `json:"visited"`
	Queue      []string  `json:"queue"`
	StartedAt  time.Time `json:"started_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CrawlState struct {
	Paused    bool                      `json:"paused"`
	Domains   map[string]*DomainProgress `json:"domains"`
	UpdatedAt time.Time                 `json:"updated_at"`
}

type Store struct {
	mu   sync.Mutex
	path string
	data CrawlState
}

func Load(path string) (*Store, error) {
	s := &Store{path: path, data: CrawlState{Domains: map[string]*DomainProgress{}}}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(b, &s.data); err != nil {
		return nil, err
	}
	if s.data.Domains == nil {
		s.data.Domains = map[string]*DomainProgress{}
	}
	return s, nil
}

func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o644)
}

func (s *Store) Pause() {
	s.mu.Lock()
	s.data.Paused = true
	s.mu.Unlock()
	_ = s.Save()
}

func (s *Store) Resume() {
	s.mu.Lock()
	s.data.Paused = false
	s.mu.Unlock()
	_ = s.Save()
}

func (s *Store) IsPaused() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Paused
}

func (s *Store) Get(domain string) *DomainProgress {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.data.Domains[domain]; ok {
		return p
	}
	p := &DomainProgress{
		Domain:    domain,
		Visited:   []string{},
		Queue:     []string{},
		StartedAt: time.Now().UTC(),
	}
	s.data.Domains[domain] = p
	return p
}

func (s *Store) IsFinished(host string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p := s.data.Domains[host]; p != nil {
		return p.Finished
	}
	return false
}

func (s *Store) ResetDomains(hosts []string) error {
	s.mu.Lock()
	for _, h := range hosts {
		delete(s.data.Domains, h)
	}
	s.mu.Unlock()
	return s.Save()
}

func (s *Store) Update(domain string, fn func(*DomainProgress)) error {
	s.mu.Lock()
	p := s.data.Domains[domain]
	if p == nil {
		p = &DomainProgress{Domain: domain, StartedAt: time.Now().UTC()}
		s.data.Domains[domain] = p
	}
	fn(p)
	p.UpdatedAt = time.Now().UTC()
	s.mu.Unlock()
	return s.Save()
}
