//
// Created by Javid Rzayev on 12.08.26.
//

package podmap

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
)

const podNamespaceLabel = "io.kubernetes.pod.namespace"
const podNameLabel = "io.kubernetes.pod.name"
const containerNameLabel = "io.kubernetes.container.name"

var ErrContainerNotFound = errors.New("container not found in CRI")

type ContainerInfo struct {
	Namespace     string
	PodName       string
	ContainerName string
}

type CRIClient struct {
	conn   *grpc.ClientConn
	client runtime.RuntimeServiceClient
}

func NewCRIClient(socket string) (*CRIClient, error) {
	if socket == "" {
		return nil, errors.New("CRI socket is empty")
	}

	conn, err := grpc.NewClient(
		"passthrough:///cri",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("create CRI client for socket %q: %w", socket, err)
	}

	client := &CRIClient{
		conn:   conn,
		client: runtime.NewRuntimeServiceClient(conn),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.client.Version(ctx, &runtime.VersionRequest{})
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("connect to CRI socket %q: %w", socket, err)
	}

	return client, nil
}

func (c *CRIClient) ContainerNames(containerID string) (ContainerInfo, error) {
	if containerID == "" {
		return ContainerInfo{}, errors.New("container ID is empty")
	}

	newCtx := context.Background()
	ctx, cancel := context.WithTimeout(newCtx, 5*time.Second)
	defer cancel()

	resp, err := c.client.ContainerStatus(ctx, &runtime.ContainerStatusRequest{
		ContainerId: containerID,
		Verbose:     false,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ContainerInfo{}, fmt.Errorf(
				"%w: %s",
				ErrContainerNotFound,
				containerID,
			)
		}

		return ContainerInfo{}, fmt.Errorf(
			"get status for container %q: %w",
			containerID,
			err,
		)
	}

	if resp == nil || resp.Status == nil {
		return ContainerInfo{}, fmt.Errorf(
			"CRI returned empty status for container %q",
			containerID,
		)
	}

	labels := resp.Status.Labels

	info := ContainerInfo{
		Namespace:     labels[podNamespaceLabel],
		PodName:       labels[podNameLabel],
		ContainerName: labels[containerNameLabel],
	}

	if info.Namespace == "" || info.PodName == "" || info.ContainerName == "" {
		return ContainerInfo{}, fmt.Errorf(
			"%w: container %q has no Kubernetes labels",
			ErrNotPod,
			containerID,
		)
	}

	return info, nil
}

func (c *CRIClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}

	return c.conn.Close()
}
