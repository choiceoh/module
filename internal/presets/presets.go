package presets

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type MasterPreset struct {
	ModuleType string `json:"module_type"`
	CellType   string `json:"cell_type"`
}

type Store struct {
	Masters        map[string]MasterPreset `json:"masters"`
	RecentProjects []string                `json:"recent_projects"`
	RecentPresets  []MasterPreset          `json:"recent_presets"`

	mu sync.Mutex
}

const (
	storeDir  = ".module-scanner"
	storeFile = "presets.json"
)

func path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, storeDir, storeFile), nil
}

func Load() (*Store, error) {
	s := &Store{
		Masters:        map[string]MasterPreset{},
		RecentProjects: []string{},
		RecentPresets:  []MasterPreset{},
	}
	p, err := path()
	if err != nil {
		return s, nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return s, nil
	}
	if err := json.Unmarshal(data, s); err != nil {
		log.Printf("presets: %s 파싱 실패, 기본값으로 진행: %v", p, err)
		s = &Store{
			Masters:        map[string]MasterPreset{},
			RecentProjects: []string{},
			RecentPresets:  []MasterPreset{},
		}
		return s, nil
	}
	if s.Masters == nil {
		s.Masters = map[string]MasterPreset{}
	}
	if s.RecentProjects == nil {
		s.RecentProjects = []string{}
	}
	if s.RecentPresets == nil {
		s.RecentPresets = []MasterPreset{}
	}
	return s, nil
}

func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// masterKey normalises a path so the same file is matched regardless
// of slash style. Full path is intentional: two files with the same
// basename in different folders should not collide.
func masterKey(masterPath string) string {
	abs, err := filepath.Abs(masterPath)
	if err != nil {
		abs = masterPath
	}
	return filepath.Clean(abs)
}

func (s *Store) GetMaster(masterPath string) (MasterPreset, bool) {
	p, ok := s.Masters[masterKey(masterPath)]
	return p, ok
}

func (s *Store) PutMaster(masterPath string, preset MasterPreset) {
	s.mu.Lock()
	s.Masters[masterKey(masterPath)] = preset
	s.mu.Unlock()
}

func (s *Store) PushRecentProject(name string) {
	if name == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]string, 0, len(s.RecentProjects)+1)
	filtered = append(filtered, name)
	for _, p := range s.RecentProjects {
		if p != name {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) > 10 {
		filtered = filtered[:10]
	}
	s.RecentProjects = filtered
}

func (s *Store) PushRecentPreset(p MasterPreset) {
	if p.ModuleType == "" && p.CellType == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]MasterPreset, 0, len(s.RecentPresets)+1)
	filtered = append(filtered, p)
	for _, e := range s.RecentPresets {
		if e.ModuleType == p.ModuleType && e.CellType == p.CellType {
			continue
		}
		filtered = append(filtered, e)
	}
	if len(filtered) > 20 {
		filtered = filtered[:20]
	}
	s.RecentPresets = filtered
}
