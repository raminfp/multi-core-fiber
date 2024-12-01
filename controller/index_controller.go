package controllers

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"multi-core-fiber/cpu"
	"multi-core-fiber/service"
	"runtime"
)

type IndexController struct {
	redisManager  *services.RedisManager
	pgManager     *services.PostgreSQLManager
	sentryManager *services.SentryManager
}

func NewIndexController(
	redisManager *services.RedisManager,
	pgManager *services.PostgreSQLManager,
	sentryManager *services.SentryManager,
) *IndexController {
	return &IndexController{
		redisManager:  redisManager,
		pgManager:     pgManager,
		sentryManager: sentryManager,
	}
}

func (ic *IndexController) StoreRequest(c *fiber.Ctx) error {
	// Get unique request ID
	requestID := int(c.Context().ID())

	// Determine current core
	currentCore := cpu.GetCurrentCore()
	//// Send Error to sentry
	//myErr := "For test Error"
	//ic.sentryManager.CaptureError(currentCore, errors.New(myErr))

	// Store request in PostgreSQL
	err := ic.pgManager.PStoreRequest(currentCore, requestID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to store request in PostgreSQL: %v", err),
		})
	}

	// Store request in Redis
	err = ic.redisManager.StoreRequest(currentCore, requestID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to store in Redis",
		})
	}

	// Return response
	return c.JSON(fiber.Map{
		"request_id": requestID,
		"core":       currentCore,
		"message":    "Request stored successfully",
	})
}

// RetrieveRequestInfo Additional method to retrieve cross-database request information
func (ic *IndexController) RetrieveRequestInfo(c *fiber.Ctx) error {
	numCores := runtime.NumCPU()

	// Retrieve from Redis
	redisResults, redisErr := ic.redisManager.RetrieveRequests(numCores)

	// Retrieve from PostgreSQL
	pgResults, pgErr := ic.pgManager.RetrieveRequests(numCores)

	// Combine results
	combinedResults := make(map[int]map[string]interface{})

	for core := 0; core < numCores; core++ {
		combinedResults[core] = map[string]interface{}{
			"redis_requests": redisResults[core],
			"pg_requests":    pgResults[core],
		}
	}

	// Handle potential errors
	if redisErr != nil || pgErr != nil {
		return c.Status(fiber.StatusPartialContent).JSON(fiber.Map{
			"results": combinedResults,
			"errors": fiber.Map{
				"redis":    redisErr,
				"postgres": pgErr,
			},
		})
	}

	return c.JSON(fiber.Map{
		"message": "Request information retrieved from both databases",
		"results": combinedResults,
	})
}
