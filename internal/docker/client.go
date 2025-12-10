// Package docker provides Docker client functionality for retrieving
// container statistics and information.
package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// StatsJSON is the stats response from Docker API
type StatsJSON struct {
	CPUStats    CPUStats            `json:"cpu_stats"`
	PreCPUStats CPUStats            `json:"precpu_stats"`
	MemoryStats MemoryStats         `json:"memory_stats"`
	Networks    map[string]NetStats `json:"networks"`
	BlkioStats  BlkioStats          `json:"blkio_stats"`
	PidsStats   PidsStats           `json:"pids_stats"`
}

// CPUStats represents CPU statistics
type CPUStats struct {
	CPUUsage    CPUUsage `json:"cpu_usage"`
	SystemUsage uint64   `json:"system_cpu_usage"`
	OnlineCPUs  uint32   `json:"online_cpus"`
}

// CPUUsage represents CPU usage details
type CPUUsage struct {
	TotalUsage  uint64   `json:"total_usage"`
	PercpuUsage []uint64 `json:"percpu_usage"`
}

// MemoryStats represents memory statistics
type MemoryStats struct {
	Usage uint64 `json:"usage"`
	Limit uint64 `json:"limit"`
}

// NetStats represents network statistics
type NetStats struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
}

// BlkioStats represents block I/O statistics
type BlkioStats struct {
	IoServiceBytesRecursive []BlkioStatEntry `json:"io_service_bytes_recursive"`
}

// BlkioStatEntry represents a single block I/O stat entry
type BlkioStatEntry struct {
	Op    string `json:"op"`
	Value uint64 `json:"value"`
}

// PidsStats represents process statistics
type PidsStats struct {
	Current uint64 `json:"current"`
}

// Client wraps the Docker client with additional functionality
type Client struct {
	cli *client.Client
}

// ContainerStats holds statistics for a single container
type ContainerStats struct {
	ID            string
	Name          string
	Image         string
	Status        string
	State         string
	CPUPercent    float64
	CPULimit      float64 // Number of CPUs (e.g., 2.0 = 2 CPUs, 0.5 = half CPU)
	MemUsage      uint64
	MemLimit      uint64
	MemPercent    float64
	NetRx         uint64
	NetTx         uint64
	BlockRead     uint64
	BlockWrite    uint64
	PIDs          uint64
	ImageSize     int64
	ContainerSize int64
	Created       time.Time
}

// SortField represents the field to sort containers by
type SortField int

const (
	SortByName SortField = iota
	SortByCPU
	SortByMemory
	SortByNetIO
	SortByBlockIO
	SortByImageSize
)

// NewClient creates a new Docker client
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cli.Ping(ctx)
	if err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	return &Client{cli: cli}, nil
}

// Close closes the Docker client connection
func (c *Client) Close() error {
	return c.cli.Close()
}

// GetContainerStats retrieves statistics for all containers
func (c *Client) GetContainerStats(ctx context.Context, showAll bool) ([]ContainerStats, error) {
	// List containers
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:  showAll,
		Size: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		return []ContainerStats{}, nil
	}

	// Get stats for each container concurrently
	var wg sync.WaitGroup
	statsChan := make(chan ContainerStats, len(containers))
	errChan := make(chan error, len(containers))

	for _, cont := range containers {
		wg.Add(1)
		go func(cont container.Summary) {
			defer wg.Done()

			stats, err := c.getContainerStats(ctx, cont)
			if err != nil {
				errChan <- err
				return
			}
			statsChan <- stats
		}(cont)
	}

	wg.Wait()
	close(statsChan)
	close(errChan)

	// Collect results
	result := make([]ContainerStats, 0, len(containers))
	for stats := range statsChan {
		result = append(result, stats)
	}

	return result, nil
}

// getContainerStats retrieves statistics for a single container
func (c *Client) getContainerStats(ctx context.Context, cont container.Summary) (ContainerStats, error) {
	stats := ContainerStats{
		ID:      cont.ID[:12],
		Name:    trimContainerName(cont.Names),
		Image:   cont.Image,
		Status:  cont.Status,
		State:   cont.State,
		Created: time.Unix(cont.Created, 0),
	}

	// Get container size
	stats.ContainerSize = cont.SizeRw

	// Get image size
	imageInfo, err := c.cli.ImageInspect(ctx, cont.ImageID)
	if err == nil {
		stats.ImageSize = imageInfo.Size
	}

	// Get CPU limit from container inspect
	containerInfo, err := c.cli.ContainerInspect(ctx, cont.ID)
	if err == nil {
		// NanoCPUs is in units of 10^-9 CPUs
		if containerInfo.HostConfig.NanoCPUs > 0 {
			stats.CPULimit = float64(containerInfo.HostConfig.NanoCPUs) / 1e9
		} else if containerInfo.HostConfig.CPUQuota > 0 && containerInfo.HostConfig.CPUPeriod > 0 {
			// CPUQuota/CPUPeriod gives the number of CPUs
			stats.CPULimit = float64(containerInfo.HostConfig.CPUQuota) / float64(containerInfo.HostConfig.CPUPeriod)
		}
		// 0 means unlimited
	}

	// Skip stats for non-running containers
	if cont.State != "running" {
		return stats, nil
	}

	// Get live stats
	statsResp, err := c.cli.ContainerStats(ctx, cont.ID, false)
	if err != nil {
		return stats, nil // Return partial stats on error
	}
	defer func() { _ = statsResp.Body.Close() }()

	var statsJSON StatsJSON
	decoder := json.NewDecoder(statsResp.Body)
	if err := decoder.Decode(&statsJSON); err != nil && err != io.EOF {
		return stats, nil
	}

	// Calculate CPU percentage
	stats.CPUPercent = calculateCPUPercent(&statsJSON)

	// Memory stats
	stats.MemUsage = statsJSON.MemoryStats.Usage
	stats.MemLimit = statsJSON.MemoryStats.Limit
	if stats.MemLimit > 0 {
		stats.MemPercent = float64(stats.MemUsage) / float64(stats.MemLimit) * 100
	}

	// Network stats
	for _, netStats := range statsJSON.Networks {
		stats.NetRx += netStats.RxBytes
		stats.NetTx += netStats.TxBytes
	}

	// Block I/O stats
	for _, blkStats := range statsJSON.BlkioStats.IoServiceBytesRecursive {
		switch blkStats.Op {
		case "read", "Read":
			stats.BlockRead += blkStats.Value
		case "write", "Write":
			stats.BlockWrite += blkStats.Value
		}
	}

	// PIDs
	stats.PIDs = statsJSON.PidsStats.Current

	return stats, nil
}

// calculateCPUPercent calculates the CPU usage percentage
func calculateCPUPercent(stats *StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)

	if systemDelta > 0 && cpuDelta > 0 {
		cpuCount := float64(stats.CPUStats.OnlineCPUs)
		if cpuCount == 0 {
			cpuCount = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
		}
		if cpuCount == 0 {
			cpuCount = 1
		}
		return (cpuDelta / systemDelta) * cpuCount * 100
	}
	return 0
}

// trimContainerName removes the leading slash from container names
func trimContainerName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	name := names[0]
	if len(name) > 0 && name[0] == '/' {
		return name[1:]
	}
	return name
}

// SortContainers sorts containers by the specified field
func SortContainers(containers []ContainerStats, field SortField, ascending bool) {
	sort.Slice(containers, func(i, j int) bool {
		var less bool
		switch field {
		case SortByName:
			less = containers[i].Name < containers[j].Name
		case SortByCPU:
			less = containers[i].CPUPercent < containers[j].CPUPercent
		case SortByMemory:
			less = containers[i].MemPercent < containers[j].MemPercent
		case SortByNetIO:
			less = (containers[i].NetRx + containers[i].NetTx) < (containers[j].NetRx + containers[j].NetTx)
		case SortByBlockIO:
			less = (containers[i].BlockRead + containers[i].BlockWrite) < (containers[j].BlockRead + containers[j].BlockWrite)
		case SortByImageSize:
			less = containers[i].ImageSize < containers[j].ImageSize
		default:
			less = containers[i].Name < containers[j].Name
		}
		if ascending {
			return less
		}
		return !less
	})
}

// GetDockerInfo retrieves Docker daemon information
func (c *Client) GetDockerInfo(ctx context.Context) (*DockerInfo, error) {
	info, err := c.cli.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker info: %w", err)
	}

	images, err := c.cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	var totalImageSize int64
	for _, img := range images {
		totalImageSize += img.Size
	}

	return &DockerInfo{
		ServerVersion:     info.ServerVersion,
		ContainersTotal:   info.Containers,
		ContainersRunning: info.ContainersRunning,
		ContainersPaused:  info.ContainersPaused,
		ContainersStopped: info.ContainersStopped,
		ImagesTotal:       len(images),
		TotalImageSize:    totalImageSize,
		MemoryTotal:       info.MemTotal,
		CPUs:              info.NCPU,
		OSType:            info.OSType,
		Architecture:      info.Architecture,
	}, nil
}

// DockerInfo holds Docker daemon information
type DockerInfo struct {
	ServerVersion     string
	ContainersTotal   int
	ContainersRunning int
	ContainersPaused  int
	ContainersStopped int
	ImagesTotal       int
	TotalImageSize    int64
	MemoryTotal       int64
	CPUs              int
	OSType            string
	Architecture      string
}
