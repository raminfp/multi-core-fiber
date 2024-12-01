package services

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// PostgreSQLManager PostgresSQLManager handles PostgreSQL connection pools for a multi-core environment
type PostgreSQLManager struct {
	pools map[int]*pgxpool.Pool
	mu    sync.Mutex
}

// NewPostgreSQLManager creates connection pools for each core
func NewPostgreSQLManager(numCores int) (*PostgreSQLManager, error) {
	manager := &PostgreSQLManager{
		pools: make(map[int]*pgxpool.Pool),
	}
	config, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %v", err)
	}

	// Customize pool configuration
	config.MaxConns = int32(numCores * 10)
	config.MinConns = int32(numCores * 5)
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 10 * time.Minute

	// Create a separate connection pool for each core
	for i := 0; i < numCores; i++ {
		pool, err := pgxpool.ConnectConfig(context.Background(), config)
		if err != nil {
			return nil, fmt.Errorf("failed to create connection pool for core %d: %v", i, err)
		}
		manager.pools[i] = pool
	}

	return manager, nil
}

// GetPoolForCore returns a connection pool for a specific core
func (pm *PostgreSQLManager) GetPoolForCore(core int) *pgxpool.Pool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.pools[core%len(pm.pools)]
}

// PStoreRequest StoreRequest stores a request number in PostgreSQL for a specific core
func (pm *PostgreSQLManager) PStoreRequest(core int, requestID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool := pm.GetPoolForCore(core)
	query := `INSERT INTO request_tracking (core, request_id) VALUES ($1, $2)`

	_, err := pool.Exec(ctx, query, core, requestID)
	return err
}

// RetrieveRequests retrieves stored requests for all cores
func (pm *PostgreSQLManager) RetrieveRequests(numCores int) (map[int][]int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := make(map[int][]int)

	for core := 0; core < numCores; core++ {
		pool := pm.GetPoolForCore(core)
		query := `SELECT request_id FROM request_tracking WHERE core = $1`

		rows, err := pool.Query(ctx, query, core)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var coreRequests []int
		for rows.Next() {
			var requestID int
			if err := rows.Scan(&requestID); err != nil {
				return nil, err
			}
			coreRequests = append(coreRequests, requestID)
		}

		results[core] = coreRequests
	}

	return results, nil
}

// Close closes all connection pools
func (pm *PostgreSQLManager) Close() {
	for _, pool := range pm.pools {
		pool.Close()
	}
}
