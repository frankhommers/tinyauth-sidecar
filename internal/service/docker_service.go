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
	log.Printf("tinyauth is healthy")
	return nil
}

// waitForHealthy waits for tinyauth to go down first, then polls until it's back up.
func (s *DockerService) waitForHealthy(timeout time.Duration) error {
	healthURL := strings.TrimRight(s.cfg.TinyauthBaseURL, "/")
	httpClient := &http.Client{Timeout: 2 * time.Second}

	deadline := time.Now().Add(timeout)

	// Phase 1: wait for tinyauth to go DOWN (max 30s of the total timeout)
	downDeadline := time.Now().Add(30 * time.Second)
	if downDeadline.After(deadline) {
		downDeadline = deadline
	}
	for time.Now().Before(downDeadline) {
		resp, err := httpClient.Get(healthURL)
		if err != nil {
			// Connection refused = it's down, move to phase 2
			log.Printf("tinyauth is down, waiting for it to come back up")
			break
		}
		resp.Body.Close()
		time.Sleep(250 * time.Millisecond)
	}

	// Phase 2: wait for tinyauth to come back UP
	for time.Now().Before(deadline) {
		resp, err := httpClient.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
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
