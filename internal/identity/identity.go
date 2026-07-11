package identity

import (
	"fmt"
	"time"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
)

type Mode int

const (
	ModePermanent Mode = iota
	ModeTemporary
)

func (m Mode) String() string {
	switch m {
	case ModePermanent:
		return "Permanent"
	case ModeTemporary:
		return "Temporary"
	default:
		return "Unknown"
	}
}

type Identity struct {
	Name      string
	Email     string
	SSHKey    string
	Mode      Mode
	CreatedAt time.Time
}

type Manager struct {
	current       *Identity
	previousState *IdentitySnapshot
	tempService   *TempService
	configStore   *config.Store
}

func NewManager() (*Manager, error) {
	store, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	tempService, err := NewTempService()
	if err != nil {
		return nil, fmt.Errorf("initializing temp service: %w", err)
	}

	manager := &Manager{
		configStore: store,
		tempService: tempService,
	}

	if store.Current != "" {
		user := store.CurrentUser()
		if user != nil {
			manager.current = &Identity{
				Name:      user.Name,
				Email:     user.Email,
				SSHKey:    user.SSHKey,
				Mode:      ModePermanent,
				CreatedAt: time.Now(),
			}
		}
	}

	return manager, nil
}

func (m *Manager) CreateTemporary(name, email string) (*Identity, error) {
	if name == "" || email == "" {
		return nil, fmt.Errorf("name and email are required")
	}

	identity := &Identity{
		Name:      name,
		Email:     email,
		Mode:      ModeTemporary,
		CreatedAt: time.Now(),
	}

	return identity, nil
}

func (m *Manager) CreatePermanent(name, email string) (*Identity, error) {
	if name == "" || email == "" {
		return nil, fmt.Errorf("name and email are required")
	}

	if err := m.configStore.AddUser(name, email); err != nil {
		return nil, fmt.Errorf("adding user to config: %w", err)
	}

	if err := config.Save(m.configStore); err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}

	identity := &Identity{
		Name:      name,
		Email:     email,
		Mode:      ModePermanent,
		CreatedAt: time.Now(),
	}

	return identity, nil
}

func (m *Manager) GetCurrent() *Identity {
	return m.current
}

func (m *Manager) SetCurrent(identity *Identity) {
	m.current = identity
}

func (m *Manager) CaptureSnapshot() error {
	snapshot := &IdentitySnapshot{
		CapturedAt: time.Now(),
	}

	snapshot.Name = git.CurrentName()
	snapshot.Email = git.CurrentEmail()

	sshCommand := git.CurrentSSHCommand()
	snapshot.SSHCommand = sshCommand

	if m.current != nil {
		snapshot.WasTemporary = m.current.Mode == ModeTemporary
	}

	m.previousState = snapshot
	return nil
}

func (m *Manager) RestoreSnapshot() error {
	if m.previousState == nil {
		git.ClearIdentity()
		m.current = nil
		return nil
	}

	snapshot := m.previousState

	if snapshot.Name != "" && snapshot.Email != "" {
		if err := git.Apply(snapshot.Name, snapshot.Email); err != nil {
			return fmt.Errorf("restoring git config: %w", err)
		}
	}

	if snapshot.SSHCommand != "" {
		if err := git.SetSSHCommand(snapshot.SSHCommand); err != nil {
			return fmt.Errorf("restoring SSH command: %w", err)
		}
	} else {
		if err := git.RemoveSSHConfig(); err != nil {
		}
	}

	if !snapshot.WasTemporary && snapshot.Name != "" {
		user := m.configStore.FindUser(snapshot.Name)
		if user != nil {
			m.current = &Identity{
				Name:      user.Name,
				Email:     user.Email,
				SSHKey:    user.SSHKey,
				Mode:      ModePermanent,
				CreatedAt: time.Now(),
			}
		}
	} else {
		m.current = nil
	}

	m.previousState = nil

	return nil
}

func (m *Manager) Activate(identity *Identity) error {
	if identity == nil {
		return fmt.Errorf("identity cannot be nil")
	}

	if err := git.Apply(identity.Name, identity.Email); err != nil {
		return fmt.Errorf("applying git config: %w", err)
	}

	if identity.SSHKey != "" {
		if err := git.ConfigureSSH(identity.SSHKey); err != nil {
			return fmt.Errorf("configuring SSH: %w", err)
		}
	}

	m.current = identity

	if identity.Mode == ModePermanent {
		if err := m.configStore.SetCurrent(identity.Name); err != nil {
			return fmt.Errorf("setting current in config: %w", err)
		}
		if err := config.Save(m.configStore); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
	}

	return nil
}

func (m *Manager) GetTempService() *TempService {
	return m.tempService
}

func (m *Manager) ReloadConfig() error {
	store, err := config.Load()
	if err != nil {
		return fmt.Errorf("reloading config: %w", err)
	}
	m.configStore = store
	return nil
}
