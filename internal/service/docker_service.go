package service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"tinyauth-sidecar/internal/config"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type DockerService struct{ cfg *config.Config }

func NewDockerService(cfg *config.Config) *DockerService { return &DockerService{cfg: cfg} }

// RestartTinyauth restarts (or signals) the tinyauth container and waits until it's healthy.
func (s *DockerService) RestartTinyauth() error {
	cli, err := client.NewClientWithOpts(
		client.WithHost("unix://"+s.cfg.DockerSocketPath),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("docker client init failed: %w", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	method := s.cfg.RestartMethod
	if method == "" {
		method = "restart"
	}

	if strings.HasPrefix(method, "signal:") {
		signal := strings.TrimPrefix(method, "signal:")
		if err := cli.ContainerKill(ctx, s.cfg.TinyauthContainerName, signal); err != nil {
			return fmt.Errorf("failed to send %s to tinyauth container %s: %w", signal, s.cfg.TinyauthContainerName, err)
		}
		log.Printf("sent %s to tinyauth container %s", signal, s.cfg.TinyauthContainerName)
	} else {
		timeout := 10
		if err := cli.ContainerRestart(ctx, s.cfg.TinyauthContainerName, container.StopOptions{Timeout: &timeout}); err != nil {
			return fmt.Errorf("failed to restart tinyauth container %s: %w", s.cfg.TinyauthContainerName, err)
		}
		log.Printf("tinyauth container %s restarted", s.cfg.TinyauthContainerName)
	}

	// Wait for tinyauth to become healthy
	if err := s.waitForHealthy(120 * time.Second); err != nil {
		return fmt.Errorf("tinyauth did not become healthy after restart: %w", err)
	}
	// Give tinyauth a moment to fully initialize all routes after healthz responds
	time.Sleep(2 * time.Second)
	log.Printf("tinyauth is healthy")
	return nil
}

// waitForHealthy polls the tinyauth HTTP endpoint until it responds with 200.
// ContainerRestart is blocking (waits for container up), but the HTTP server
// inside may need a moment to start accepting connections.
func (s *DockerService) waitForHealthy(timeout time.Duration) error {
	// Use the health endpoint instead of / to ensure the API is actually ready.
	// The SPA at / returns 200 before the backend is fully initialized.
	healthURL := strings.TrimRight(s.cfg.TinyauthBaseURL, "/") + "/api/healthz"
	httpClient := &http.Client{Timeout: 2 * time.Second}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := httpClient.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			// Accept any 2xx as healthy (some versions return 200, others 204)
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", healthURL)
}

// IsTinyauthRunning checks if the tinyauth container is running.
func (s *DockerService) IsTinyauthRunning() (bool, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost("unix://"+s.cfg.DockerSocketPath),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return false, err
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := cli.ContainerInspect(ctx, s.cfg.TinyauthContainerName)
	if err != nil {
		return false, err
	}
	return info.State != nil && info.State.Running, nil
}
