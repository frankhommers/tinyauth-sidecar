package service

import (
	"context"
	"log"
	"time"

	"tinyauth-usermanagement/internal/config"

	"github.com/docker/docker/client"
)

type DockerService struct{ cfg *config.Config }

func NewDockerService(cfg *config.Config) *DockerService { return &DockerService{cfg: cfg} }

// RestartTinyauth kills the tinyauth container with SIGTERM after a short delay.
// The delay ensures the API response is sent back to the user first.
// Docker's restart policy (restart: unless-stopped) brings it back up quickly.
func (s *DockerService) RestartTinyauth() {
	go func() {
		time.Sleep(500 * time.Millisecond)

		cli, err := client.NewClientWithOpts(
			client.WithHost("unix://"+s.cfg.DockerSocketPath),
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			log.Printf("docker client init failed: %v", err)
			return
		}
		defer cli.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := cli.ContainerKill(ctx, s.cfg.TinyauthContainerName, "SIGTERM"); err != nil {
			log.Printf("failed to kill tinyauth container %s: %v", s.cfg.TinyauthContainerName, err)
		} else {
			log.Printf("tinyauth container %s killed (SIGTERM), waiting for restart policy", s.cfg.TinyauthContainerName)
		}
	}()
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
