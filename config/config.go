package config

import (
	"encoding/json"
	"os"
	"sync"
)

type Config struct {
	ListenAddr     string
	ServerAddr     string
	Headless       bool
	ReconnectDelay int
}

// ModuleSettings stores per-module enabled state and key/value settings
type ModuleSettings struct {
	mu      sync.RWMutex
	path    string
	Modules map[string]*ModuleEntry `json:"modules"`
}

type ModuleEntry struct {
	Enabled  bool                   `json:"enabled"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}

func LoadSettings(path string) *ModuleSettings {
	s := &ModuleSettings{
		path:    path,
		Modules: make(map[string]*ModuleEntry),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	json.Unmarshal(data, s)
	return s
}

func (s *ModuleSettings) Save() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(s.path, data, 0600)
}

func (s *ModuleSettings) IsEnabled(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if e, ok := s.Modules[name]; ok {
		return e.Enabled
	}
	return false
}

func (s *ModuleSettings) SetEnabled(name string, enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Modules[name]; !ok {
		s.Modules[name] = &ModuleEntry{Settings: make(map[string]interface{})}
	}
	s.Modules[name].Enabled = enabled
}

func (s *ModuleSettings) Get(name, key string, def interface{}) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if e, ok := s.Modules[name]; ok {
		if v, ok := e.Settings[key]; ok {
			return v
		}
	}
	return def
}

func (s *ModuleSettings) Set(name, key string, val interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Modules[name]; !ok {
		s.Modules[name] = &ModuleEntry{Settings: make(map[string]interface{})}
	}
	s.Modules[name].Settings[key] = val
}
