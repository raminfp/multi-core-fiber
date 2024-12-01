package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"multi-core-fiber/controller"
	"multi-core-fiber/route"
	"multi-core-fiber/service"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, using default configuration")
	}
	// Utilize all available cores
	numCores := runtime.NumCPU()
	runtime.GOMAXPROCS(numCores)

	// Create Redis connection manager
	redisManager := services.NewRedisManager(numCores)
	// Create PostgreSQL connection manager
	pgManager, err := services.NewPostgreSQLManager(numCores)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL manager: %v", err)
	}
	// Create Sentry connection manager
	sentryManager, err := services.NewSentryManager(numCores)
	if err != nil {
		log.Fatalf("Failed to create Sentry manager: %v", err)
	}
	defer pgManager.Close()
	defer redisManager.Close()
	// Create Fiber app
	app := fiber.New(fiber.Config{
		ServerHeader: "Multi-Core Redis Demo",
	})
	// Create controller
	indexController := controllers.NewIndexController(
		redisManager,
		pgManager,
		sentryManager)

	// Setup routes
	routes.SetupIndexRoutes(app, indexController)

	// Startup logging
	fmt.Printf("Server Starting with %d Cores\n", numCores)
	fmt.Println("Cores Available:", runtime.NumCPU())

	// Start server
	go func() {
		if err := app.Listen(":3001"); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Handle graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	log.Println("Gracefully shutting down...")
	app.Shutdown()
}
