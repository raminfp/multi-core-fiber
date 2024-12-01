package services

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"io"
	"sync"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioManager handles interactions with Minio for a multi-core environment
type MinioManager struct {
	clients map[int]*minio.Client
	bucket  string
	mu      sync.Mutex
}

// MinioConfig contains configuration for Minio connection
type MinioConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	UseSSL          bool
}

// NewMinioManager creates Minio clients for each core
func NewMinioManager(config MinioConfig, numCores int) (*MinioManager, error) {
	manager := &MinioManager{
		clients: make(map[int]*minio.Client),
		bucket:  config.Bucket,
	}

	for i := 0; i < numCores; i++ {
		// Create Minio client
		client, err := minio.New(config.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
			Secure: config.UseSSL,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Minio client for core %d: %v", i, err)
		}

		manager.clients[i] = client
	}

	return manager, nil
}

// GetClientForCore returns the Minio client for a specific core
func (mm *MinioManager) GetClientForCore(core int) *minio.Client {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	return mm.clients[core%len(mm.clients)]
}

// UploadFile uploads a file to Minio for a specific core
func (mm *MinioManager) UploadFile(ctx context.Context, core int, objectName string, file io.Reader, size int64, contentType string) error {
	client := mm.GetClientForCore(core)

	_, err := client.PutObject(ctx, mm.bucket, objectName, file, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file on core %d: %w", core, err)
	}
	return nil
}

// DownloadFile downloads a file from Minio for a specific core
func (mm *MinioManager) DownloadFile(ctx context.Context, core int, objectName string) (io.ReadCloser, error) {
	client := mm.GetClientForCore(core)

	object, err := client.GetObject(ctx, mm.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download file on core %d: %w", core, err)
	}
	return object, nil
}

// DeleteFile deletes a file from Minio for a specific core
func (mm *MinioManager) DeleteFile(ctx context.Context, core int, objectName string) error {
	client := mm.GetClientForCore(core)

	err := client.RemoveObject(ctx, mm.bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file on core %d: %w", core, err)
	}
	return nil
}

// Close closes all Minio clients
func (mm *MinioManager) Close() {
	// Perform any necessary cleanup
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.clients = make(map[int]*minio.Client)
}
