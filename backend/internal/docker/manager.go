package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"mtproxy-manager/internal/config"
	"mtproxy-manager/internal/database"
)

type Manager struct {
	cli *client.Client
	cfg *config.Config
	db  *database.DB
}

func NewManager(cfg *config.Config, db *database.DB) (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}

	return &Manager{cli: cli, cfg: cfg, db: db}, nil
}

func (m *Manager) Close() error {
	return m.cli.Close()
}

func (m *Manager) EnsureImage(ctx context.Context) error {
	reader, err := m.cli.ImagePull(ctx, m.cfg.MTGImage, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull image: %w", err)
	}
	defer reader.Close()
	io.Copy(io.Discard, reader)
	return nil
}

func (m *Manager) AllocatePort() (int, error) {
	return m.allocatePortInRange(m.cfg.PortMin, m.cfg.PortMax)
}

func (m *Manager) allocatePortInRange(min, max int) (int, error) {
	usedPorts, err := m.db.GetUsedPorts()
	if err != nil {
		return 0, err
	}

	for port := min; port <= max; port++ {
		if !usedPorts[port] {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no free ports available in range %d-%d", min, max)
}

func (m *Manager) GetServerIP() string {
	return m.cfg.ServerIP
}
