package main

import (
	// Import local packages for DB, middleware, and route definitions
	"golang-Hotel_Management/database"
	"golang-Hotel_Management/middleware"
	"golang-Hotel_Management/routes"
	"os"

	// Gin: HTTP web framework
	"github.com/gin-gonic/gin"

	// MongoDB Go driver
	"go.mongodb.org/mongo-driver/mongo"
)

// Initialize a collection variable pointing to "food" collection in MongoDB
var foodCollection *mongo.Collection = database.OpenCollection(database.Client, "food")

func main() {
	// Load port from environment variable, default to 8000 if not set
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	// Create a new Gin router instance
	router := gin.New()

	// Use Gin's built-in logger middleware for logging HTTP requests
	router.Use(gin.Logger())

	// Setup public user authentication routes (e.g., login, signup)
	routes.UserRoutes(router)

	// Apply JWT-based authentication middleware to secure subsequent routes
	router.Use(middleware.Authentication())

	// Register all the API route groups
	routes.FoodRoutes(router)
	routes.MenuRoutes(router)
	routes.TableRoutes(router)
	routes.OrderRoutes(router)
	routes.OrderItemRoutes(router)
	routes.InvoiceRoutes(router)

	// Start the server on the specified port
	router.Run(":" + port)
}
