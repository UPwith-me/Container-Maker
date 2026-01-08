package environment

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	stateFileName   = "environments.json"
	activeEnvFile   = "active-environment"
	envStateDirName = ".cm-environments"
)

// FileStateStore implements StateStore using local filesystem
type FileStateStore struct {
	baseDir      string
	environments map[string]*Environment
	activeEnv    string
	mu           sync.RWMutex
}

// stateData represents the JSON structure for persistence
type stateData struct {
	Version      int                     `json:"version"`
	ActiveEnv    string                  `json:"active_env,omitempty"`
	Environments map[string]*Environment `json:"environments"`
	LastSync     time.Time               `json:"last_sync"`
}

// NewFileStateStore creates a new file-based state store
func NewFileStateStore() (*FileStateStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, WrapError(err, "STATE_INIT_ERROR", "failed to get home directory")
	}

	baseDir := filepath.Join(home, ".cm", envStateDirName)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, WrapError(err, "STATE_INIT_ERROR", "failed to create state directory")
	}

	store := &FileStateStore{
		baseDir:      baseDir,
		environments: make(map[string]*Environment),
	}

	// Load existing state
	if err := store.load(); err != nil {
		// If file doesn't exist, that's fine
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return store, nil
}

// getStatePath returns the path to the state file
func (s *FileStateStore) getStatePath() string {
	return filepath.Join(s.baseDir, stateFileName)
}

// load reads the state from disk
func (s *FileStateStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.getStatePath())
	if err != nil {
		return err
	}

	var state stateData
	if err := json.Unmarshal(data, &state); err != nil {
		return WrapError(err, "STATE_PARSE_ERROR", "failed to parse state file")
	}

	s.environments = state.Environments
	s.activeEnv = state.ActiveEnv

	if s.environments == nil {
		s.environments = make(map[string]*Environment)
	}

	return nil
}

// persist writes the state to disk
func (s *FileStateStore) persist() error {
	state := stateData{
		Version:      1,
		ActiveEnv:    s.activeEnv,
		Environments: s.environments,
		LastSync:     time.Now(),
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return WrapError(err, "STATE_SERIALIZE_ERROR", "failed to serialize state")
	}

	tmpFile := s.getStatePath() + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return WrapError(err, "STATE_WRITE_ERROR", "failed to write state file")
	}

	// Atomic rename
	if err := os.Rename(tmpFile, s.getStatePath()); err != nil {
		os.Remove(tmpFile) // Clean up
		return WrapError(err, "STATE_WRITE_ERROR", "failed to finalize state file")
	}

	return nil
}

// Save saves an environment to the store
func (s *FileStateStore) Save(env *Environment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if env == nil || env.ID == "" {
		return ErrInvalidConfig.WithSuggestion("environment must have valid ID")
	}

	env.UpdatedAt = time.Now()
	s.environments[env.ID] = env

	return s.persist()
}

// Load loads an environment by ID
func (s *FileStateStore) Load(id string) (*Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	env, ok := s.environments[id]
	if !ok {
		return nil, ErrEnvironmentNotFound.WithEnv(id, "")
	}

	return env, nil
}

// LoadByName loads an environment by name
func (s *FileStateStore) LoadByName(name string) (*Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, env := range s.environments {
		if env.Name == name {
			return env, nil
		}
	}

	return nil, ErrEnvironmentNotFound.WithEnv("", name)
}

// Delete removes an environment from the store
func (s *FileStateStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.environments[id]; !ok {
		return ErrEnvironmentNotFound.WithEnv(id, "")
	}

	delete(s.environments, id)

	// Clear active if it was the deleted env
	if s.activeEnv == id {
		s.activeEnv = ""
	}

	return s.persist()
}

// List returns all environments
func (s *FileStateStore) List() ([]*Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Environment, 0, len(s.environments))
	for _, env := range s.environments {
		result = append(result, env)
	}

	return result, nil
}

// SetActive sets the active environment
func (s *FileStateStore) SetActive(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify environment exists
	if id != "" {
		if _, ok := s.environments[id]; !ok {
			return ErrEnvironmentNotFound.WithEnv(id, "")
		}
	}

	s.activeEnv = id
	return s.persist()
}

// GetActive returns the active environment ID
func (s *FileStateStore) GetActive() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.activeEnv, nil
}

// Sync reconciles state with actual Docker state
func (s *FileStateStore) Sync() error {
	// This will be implemented to check actual container states
	// and update the store accordingly
	return nil
}

// GetByContainerID finds an environment by its container ID
func (s *FileStateStore) GetByContainerID(containerID string) (*Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, env := range s.environments {
		if env.ContainerID == containerID {
			return env, nil
		}
	}

	return nil, ErrEnvironmentNotFound
}

// GetByName finds an environment by name (convenience method)
func (s *FileStateStore) GetByName(name string) (*Environment, error) {
	return s.LoadByName(name)
}

// FindByStatus returns all environments with the given status
func (s *FileStateStore) FindByStatus(status EnvironmentStatus) ([]*Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Environment
	for _, env := range s.environments {
		if env.Status == status {
			result = append(result, env)
		}
	}

	return result, nil
}

// FindByProject returns all environments for a project directory
func (s *FileStateStore) FindByProject(projectDir string) ([]*Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Environment
	for _, env := range s.environments {
		if env.ProjectDir == projectDir {
			result = append(result, env)
		}
	}

	return result, nil
}

// Count returns the total number of environments
func (s *FileStateStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.environments)
}

// UpdateStatus updates just the status of an environment
func (s *FileStateStore) UpdateStatus(id string, status EnvironmentStatus, msg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	env, ok := s.environments[id]
	if !ok {
		return ErrEnvironmentNotFound.WithEnv(id, "")
	}

	env.Status = status
	env.StatusMsg = msg
	env.UpdatedAt = time.Now()

	return s.persist()
}

// UpdateLastUsed updates the last used timestamp
func (s *FileStateStore) UpdateLastUsed(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	env, ok := s.environments[id]
	if !ok {
		return ErrEnvironmentNotFound.WithEnv(id, "")
	}

	env.LastUsedAt = time.Now()
	env.UpdatedAt = time.Now()

	return s.persist()
}

// ExportState exports the state for backup
func (s *FileStateStore) ExportState() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := stateData{
		Version:      1,
		ActiveEnv:    s.activeEnv,
		Environments: s.environments,
		LastSync:     time.Now(),
	}

	return json.MarshalIndent(state, "", "  ")
}

// ImportState imports state from backup
func (s *FileStateStore) ImportState(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var state stateData
	if err := json.Unmarshal(data, &state); err != nil {
		return WrapError(err, "STATE_IMPORT_ERROR", "failed to parse import data")
	}

	s.environments = state.Environments
	s.activeEnv = state.ActiveEnv

	if s.environments == nil {
		s.environments = make(map[string]*Environment)
	}

	return s.persist()
}

// String implements fmt.Stringer for debugging
func (s *FileStateStore) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return fmt.Sprintf("FileStateStore{path=%s, envs=%d, active=%s}",
		s.baseDir, len(s.environments), s.activeEnv)
}
