package testutils

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

var (
	// ErrClientError is returned, when the client should return an error
	ErrClientError = errors.New("client failure")

	// ErrAlreadyRunning is returned, if a container which should be started is already running
	ErrAlreadyRunning = errors.New("container already running")
)

// MockDCli simulates a docker client for testing purposes
type MockDCli struct {
	running         map[string]bool
	containers      map[string]bool
	images          map[string]bool
	err             bool
	idNotExist      bool
	dockerIsOffline bool
}

// SetError sets the err property of MockDCli to be true, causing the next instruction to return an error
func (d *MockDCli) SetError() {
	d.err = true
}

// SetDockerOffline simulates a non responding docker daemon
func (d *MockDCli) SetDockerOffline() {
	d.dockerIsOffline = true
}

// SetIDNotExisting simulates a non existing docker container id
func (d *MockDCli) SetIDNotExisting() {
	d.idNotExist = true
}

func (d *MockDCli) produceError() bool {
	if d.err {
		d.err = false
		return true
	}
	return false
}

// IsRunning checks if a mocked container is running
func (d *MockDCli) IsRunning(container string) bool {
	return d.running[container]
}

// CreateMockImage creates a mock image
func (d *MockDCli) CreateMockImage(image string) {
	d.images[image] = true
}

// ContainerStart is
func (d *MockDCli) ContainerStart(ctx context.Context, container string, options types.ContainerStartOptions) error {
	if d.produceError() {
		return ErrClientError
	}
	if !d.IsRunning(container) {
		d.running[container] = true
		return nil
	}
	return ErrAlreadyRunning
}

// ContainerKill is
func (d *MockDCli) ContainerKill(ctx context.Context, container string, signal string) error {
	if d.produceError() || !d.IsRunning(container) {
		return ErrClientError
	}
	d.running[container] = false
	return nil
}

// ContainerExecCreate is
func (d *MockDCli) ContainerExecCreate(ctx context.Context, container string, config types.ExecConfig) (string, error) {
	if d.produceError() || !d.IsRunning(container) {
		return "", ErrClientError
	}
	return strings.Join(config.Cmd, " "), nil
}

// ContainerCreate is
func (d *MockDCli) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
	if d.produceError() || d.dockerIsOffline {
		return container.ContainerCreateCreatedBody{}, ErrClientError
	}

	d.containers[containerName] = true

	return container.ContainerCreateCreatedBody{
		ID: containerName,
	}, nil
}

// ContainerRename renames a container with a given ID
func (d *MockDCli) ContainerRename(ctx context.Context, containerID, newContainerName string) error {
	if d.produceError() || !d.containers[containerID] || d.idNotExist {
		return ErrClientError
	}

	return nil
}

// ContainerRemove removes a container with a given ID
func (d *MockDCli) ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error {
	if d.produceError() || !d.containers[containerID] {
		return ErrClientError
	}

	d.containers[containerID] = false

	return nil
}

// ImageInspectWithRaw is
func (d *MockDCli) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	if d.images[imageID] {
		return types.ImageInspect{}, nil, nil
	}

	return types.ImageInspect{}, nil, fmt.Errorf("Image does not exist")
}

// IsErrImageNotFound returns true if the error means the image was not found
func (d *MockDCli) IsErrImageNotFound(err error) bool {
	if err != nil {
		return true
	}

	return false
}

// NewMockDCli returns a new instance of MockDCli
func NewMockDCli() *MockDCli {
	return &MockDCli{
		running:    make(map[string]bool),
		containers: make(map[string]bool),
		images:     make(map[string]bool),
	}
}
