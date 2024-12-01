package routes

import (
	"github.com/gofiber/fiber/v2"
	"multi-core-fiber/controller"
)

func SetupIndexRoutes(app *fiber.App, indexController *controllers.IndexController) {
	// Endpoint to store request number in Redis
	app.Get("/store-request", indexController.StoreRequest)

	// Endpoint to retrieve stored requests
	app.Get("/retrieve-requests", indexController.RetrieveRequestInfo)

}
