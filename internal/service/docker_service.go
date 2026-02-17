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

	// Get StartedAt before restart so we can verify the container actually restarted
	startedBefore, _ := s.getStartedAt(cli, ctx)

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
		log.Printf("tinyauth container %s restart command completed", s.cfg.TinyauthContainerName)
	}

	// Phase 1: verify the container actually restarted by checking StartedAt changed
	if startedBefore != "" {
		if err := s.waitForNewStart(cli, ctx, startedBefore); err != nil {
			log.Printf("[restart] warning: %v", err)
			// Continue anyway — maybe it restarted too fast to catch
		}
	}

	// Phase 2: wait for HTTP endpoint to be ready
	if err := s.waitForHealthy(120 * time.Second); err != nil {
		return fmt.Errorf("tinyauth did not become healthy after restart: %w", err)
	}
	log.Printf("tinyauth is healthy")
	return nil
}

// getStartedAt returns the container's StartedAt timestamp.
func (s *DockerService) getStartedAt(cli *client.Client, ctx context.Context) (string, error) {
	info, err := cli.ContainerInspect(ctx, s.cfg.TinyauthContainerName)
	if err != nil {
		return "", err
	}
	if info.State != nil {
		return info.State.StartedAt, nil
	}
	return "", nil
}

// waitForNewStart polls until the container's StartedAt differs from the original value.
func (s *DockerService) waitForNewStart(cli *client.Client, ctx context.Context, oldStartedAt string) error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		current, err := s.getStartedAt(cli, ctx)
		if err == nil && current != "" && current != oldStartedAt {
			log.Printf("tinyauth container restarted (was: %s, now: %s)", oldStartedAt, current)
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("container StartedAt did not change within 30s")
}

// waitForHealthy polls the tinyauth health endpoint until it responds with 2xx.
// If an external URL is configured, it first waits for the internal endpoint,
// then also waits for the external one (to confirm Traefik has re-discovered the container).
func (s *DockerService) waitForHealthy(timeout time.Duration) error {
	internalURL := strings.TrimRight(s.cfg.TinyauthBaseURL, "/") + "/api/healthz"

	if err := s.pollHealthEndpoint(internalURL, timeout); err != nil {
		return err
	}

	if s.cfg.TinyauthExternalURL != "" {
		externalURL := strings.TrimRight(s.cfg.TinyauthExternalURL, "/") + "/api/healthz"
		log.Printf("internal health OK, waiting for external endpoint: %s", externalURL)
		if err := s.pollHealthEndpoint(externalURL, timeout); err != nil {
			return fmt.Errorf("external health check failed: %w", err)
		}
		log.Printf("external health OK")
	}

	return nil
}

// pollHealthEndpoint polls a URL until it responds with 2xx or the timeout expires.
func (s *DockerService) pollHealthEndpoint(url string, timeout time.Duration) error {
	httpClient := &http.Client{Timeout: 2 * time.Second}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := httpClient.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", url)
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
