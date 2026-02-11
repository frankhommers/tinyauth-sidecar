package service

import (
	"context"
	"log"
	"time"

	"tinyauth-usermanagement/internal/config"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type DockerService struct{ cfg config.Config }

func NewDockerService(cfg config.Config) *DockerService { return &DockerService{cfg: cfg} }

func (s *DockerService) RestartTinyauth() {
	cli, err := client.NewClientWithOpts(
		client.WithHost("unix://"+s.cfg.DockerSocketPath),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		log.Printf("docker client init failed: %v", err)
		return
	}
	defer cli.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	t := 10
	if err := cli.ContainerRestart(ctx, s.cfg.TinyauthContainerName, container.StopOptions{Timeout: &t}); err != nil {
		log.Printf("failed to restart tinyauth container %s: %v", s.cfg.TinyauthContainerName, err)
	}
}
