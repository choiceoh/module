package presets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type MasterPreset struct {
	ModuleType string `json:"module_type"`
	CellType   string `json:"cell_type"`
}

type Settings struct {
	VLLMBaseURL     string `json:"vllm_base_url"`
	VLLMModel       string `json:"vllm_model"`
	UseVLLMFallback bool   `json:"use_vllm_fallback"`
}

type Store struct {
	Masters        map[string]MasterPreset `json:"masters"`
	RecentProjects []string                `json:"recent_projects"`
	Settings       Settings                `json:"settings"`

	mu sync.Mutex
}

const (
	storeDir  = ".module-scanner"
	storeFile = "presets.json"
)

func defaultSettings() Settings {
	return Settings{
		VLLMBaseURL:     "https://sensation-glider-unknotted.ngrok-free.dev/v1",
		VLLMModel:       "gemma4",
		UseVLLMFallback: false,
	}
}

func path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, storeDir, storeFile), nil
}

func Load() (*Store, error) {
	s := &Store{Masters: map[string]MasterPreset{}, Settings: defaultSettings()}
	p, err := path()
	if err != nil {
		return s, nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return s, nil
	}
	_ = json.Unmarshal(data, s)
	if s.Masters == nil {
		s.Masters = map[string]MasterPreset{}
	}
	if s.Settings.VLLMBaseURL == "" {
		s.Settings.VLLMBaseURL = defaultSettings().VLLMBaseURL
	}
	if s.Settings.VLLMModel == "" {
		s.Settings.VLLMModel = defaultSettings().VLLMModel
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

func (s *Store) GetMaster(masterPath string) (MasterPreset, bool) {
	key := filepath.Base(masterPath)
	p, ok := s.Masters[key]
	return p, ok
}

func (s *Store) PutMaster(masterPath string, preset MasterPreset) {
	s.mu.Lock()
	key := filepath.Base(masterPath)
	s.Masters[key] = preset
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

func (s *Store) GetSettings() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Settings
}

func (s *Store) PutSettings(v Settings) {
	s.mu.Lock()
	s.Settings = v
	s.mu.Unlock()
}
