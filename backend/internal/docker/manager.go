package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
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

func (m *Manager) GenerateSecret(ctx context.Context, domain string) (string, error) {
	resp, err := m.cli.ContainerCreate(ctx, &container.Config{
		Image: m.cfg.MTGImage,
		Cmd:   []string{"generate-secret", domain, "--hex"},
	}, nil, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("create secret container: %w", err)
	}
	defer m.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})

	if err := m.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("start secret container: %w", err)
	}

	waitCh, errCh := m.cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case <-waitCh:
	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("wait secret container: %w", err)
		}
	case <-time.After(30 * time.Second):
		return "", fmt.Errorf("timeout generating secret")
	}

	logs, err := m.cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		return "", fmt.Errorf("read secret logs: %w", err)
	}
	defer logs.Close()

	var buf bytes.Buffer
	io.Copy(&buf, logs)

	secret := strings.TrimSpace(stripDockerLogHeaders(buf.Bytes()))
	if secret == "" {
		return "", fmt.Errorf("empty secret generated")
	}

	return secret, nil
}

func stripDockerLogHeaders(data []byte) string {
	var result []byte
	for len(data) > 0 {
		if len(data) < 8 {
			result = append(result, data...)
			break
		}
		size := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])
		data = data[8:]
		if size > len(data) {
			size = len(data)
		}
		result = append(result, data[:size]...)
		data = data[size:]
	}
	return string(result)
}

func (m *Manager) AllocatePort() (int, error) {
	return m.allocatePortInRange(m.cfg.PortMin, m.cfg.PortMax)
}

func (m *Manager) AllocateSOCKS5Port() (int, error) {
	return m.allocatePortInRange(m.cfg.Socks5PortMin, m.cfg.Socks5PortMax)
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

func (m *Manager) CreateAndStartProxy(ctx context.Context, port int, secret, containerName string) (string, error) {
	hostPort := fmt.Sprintf("%d", port)
	containerPort := nat.Port("443/tcp")

	resp, err := m.cli.ContainerCreate(ctx, &container.Config{
		Image: m.cfg.MTGImage,
		Cmd:   []string{"simple-run", "0.0.0.0:443", secret},
		ExposedPorts: nat.PortSet{
			containerPort: struct{}{},
		},
		Labels: map[string]string{
			"managed-by": "mtproxy-manager",
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			containerPort: []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: hostPort},
			},
		},
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}

	if err := m.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		m.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return "", fmt.Errorf("start container: %w", err)
	}

	return resp.ID, nil
}

func (m *Manager) StopProxy(ctx context.Context, containerID string) error {
	timeout := 10
	return m.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (m *Manager) StartProxy(ctx context.Context, containerID string) error {
	return m.cli.ContainerStart(ctx, containerID, container.StartOptions{})
}

func (m *Manager) RemoveProxy(ctx context.Context, containerID string) error {
	return m.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
}

func (m *Manager) CreateAndStartSOCKS5Proxy(ctx context.Context, port int, user, pass, containerName string) (string, error) {
	hostPort := fmt.Sprintf("%d", port)
	containerPort := nat.Port("1080/tcp")
	// GOST: gost -L socks5://user:pass@:1080
	listenAddr := fmt.Sprintf("socks5://%s:%s@:1080", user, pass)

	resp, err := m.cli.ContainerCreate(ctx, &container.Config{
		Image: m.cfg.GostImage,
		Cmd:   []string{"-L", listenAddr},
		ExposedPorts: nat.PortSet{
			containerPort: struct{}{},
		},
		Labels: map[string]string{
			"managed-by": "mtproxy-manager",
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			containerPort: []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: hostPort},
			},
		},
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("create socks5 container: %w", err)
	}

	if err := m.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		m.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return "", fmt.Errorf("start socks5 container: %w", err)
	}

	return resp.ID, nil
}

func (m *Manager) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	info, err := m.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "error", nil
	}
	if info.State.Running {
		return "running", nil
	}
	return "stopped", nil
}

func (m *Manager) GetServerIP() string {
	return m.cfg.ServerIP
}
