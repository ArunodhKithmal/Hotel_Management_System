package routes

import (
	// Importing the controller package where handler functions are defined
	controller "golang-Hotel_Management/controllers"

	// Importing the Gin framework
	"github.com/gin-gonic/gin"
)

// FoodRoutes defines all the API routes related to food operations
func FoodRoutes(incomingRoutes *gin.Engine) {
	// GET endpoint to retrieve a list of all food items
	incomingRoutes.GET("/foods", controller.GetFoods())

	// GET endpoint to retrieve details of a specific food item by its ID
	incomingRoutes.GET("/foods/:food_id", controller.GetFood())

	// POST endpoint to create a new food item
	incomingRoutes.POST("/foods", controller.CreateFood())

	// PATCH endpoint to update an existing food item by its food_id
	// NOTE: It should be "/foods/:food_id" instead of "/foods/food_id"
	//       to correctly capture the path parameter
	incomingRoutes.PATCH("/foods/:food_id", controller.UpdateFood())
}
