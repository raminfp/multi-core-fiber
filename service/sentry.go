package services

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	"os"
	"sync"
	"time"
)

// SentryManager handles interactions with Sentry for a multi-core environment
type SentryManager struct {
	clients map[int]*sentry.Client
	mu      sync.Mutex
}

// NewSentryManager creates Sentry clients for each core
func NewSentryManager(numCores int) (*SentryManager, error) {
	manager := &SentryManager{
		clients: make(map[int]*sentry.Client),
	}

	for i := 0; i < numCores; i++ {
		dsn := os.Getenv("SENTRY_DSN")
		fmt.Println(dsn)
		client, err := sentry.NewClient(sentry.ClientOptions{
			Dsn: dsn,
			//Debug: true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Sentry client for core %d: %v", i, err)
		}

		manager.clients[i] = client
	}

	return manager, nil
}

// GetClientForCore returns the Sentry client for a specific core
func (sm *SentryManager) GetClientForCore(core int) *sentry.Client {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.clients[core%len(sm.clients)]
}

// CaptureError captures an error event in Sentry for a specific core
func (sm *SentryManager) CaptureError(core int, err error) {
	client := sm.GetClientForCore(core)
	if client == nil {
		fmt.Println("Sentry client is nil")
		return
	}
	// Create an event with tags
	event := sentry.NewEvent()
	event.Message = err.Error()
	event.Level = sentry.LevelError
	event.Tags = map[string]string{
		"core": fmt.Sprintf("%d", core),
	}

	go func() {
		eventID := client.CaptureEvent(event, &sentry.EventHint{}, nil)
		if eventID != nil && *eventID == "" {
			fmt.Printf("Failed to send error to Sentry for core %d: %v\n", core, err)
		}
		// Flush with a shorter timeout
		if ok := client.Flush(time.Second); !ok {
			fmt.Printf("Flush may not have completed for core %d\n", core)
		}
	}()
}

// CaptureMessage captures a generic message to Sentry for a specific core
func (sm *SentryManager) CaptureMessage(core int, message string) {
	client := sm.GetClientForCore(core)
	event := sentry.NewEvent()
	event.Message = message
	event.Level = sentry.LevelInfo
	event.Tags = map[string]string{
		"core": fmt.Sprintf("%d", core),
	}
	//fmt.Println(message)
	client.CaptureEvent(event, &sentry.EventHint{}, nil)
}

// Close closes all Sentry clients (if needed, Sentry's SDK is typically non-blocking)
func (sm *SentryManager) Close() {
	for _, client := range sm.clients {
		client.Flush(2 * time.Second)
	}
}
