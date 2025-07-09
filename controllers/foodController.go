package controller

import (
	"context"
	"fmt"
	"golang-Hotel_Management/database"
	"golang-Hotel_Management/models"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Connect to the "food" collection in MongoDB
var foodCollection *mongo.Collection = database.OpenCollection(database.Client, "food")
var validate = validator.New()

// / GetFoods returns a paginated list of all food items in the collection.
// It uses MongoDB's aggregation framework to return total count and sliced records.
func GetFoods() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Context with 100-second timeout to limit MongoDB operations
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Get 'recordPerPage' from query, default to 10 if invalid or not provided
		recordPerPage, err := strconv.Atoi(c.Query("recordPerPage"))
		if err != nil || recordPerPage < 1 {
			recordPerPage = 10
		}

		// Get 'page' from query, default to 1 if invalid
		page, err := strconv.Atoi(c.Query("page"))
		if err != nil || page < 1 {
			page = 1
		}

		// Calculate start index for pagination
		startIndex := (page - 1) * recordPerPage
		startIndex, err = strconv.Atoi(c.Query("startIndex")) // Optional override

		// MongoDB Aggregation Stages

		// 1. Match stage (currently matches all documents)
		matchStage := bson.D{{Key: "$match", Value: bson.D{}}}

		// 2. Group all documents and push them into a "data" array while counting
		groupStage := bson.D{{
			Key: "$group", Value: bson.D{
				{Key: "_id", Value: bson.D{{Key: "_id", Value: "null"}}},
				{Key: "total_count", Value: bson.D{{Key: "$sum", Value: 1}}},
				{Key: "data", Value: bson.D{{Key: "$push", Value: "$$ROOT"}}},
			},
		}}

		// 3. Project the desired page slice from the "data" array
		projectStage := bson.D{{
			Key: "$project", Value: bson.D{
				{Key: "_id", Value: 0},
				{Key: "total_count", Value: 1},
				{Key: "food_items", Value: bson.D{{Key: "$slice", Value: []interface{}{"$data", startIndex, recordPerPage}}}},
			},
		}}

		// Execute aggregation pipeline
		result, err := foodCollection.Aggregate(ctx, mongo.Pipeline{
			matchStage, groupStage, projectStage,
		})
		defer cancel()

		// Handle aggregation error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing food items"})
			return
		}

		// Decode results into an array
		var allFoods []bson.M
		if err = result.All(ctx, &allFoods); err != nil {
			log.Fatal(err)
		}

		// Return the first (and only) item in response: total_count + food_items
		c.JSON(http.StatusOK, allFoods[0])
	}
}

// GetFood retrieves a single food item by its ID
// GetFood handles HTTP GET requests to retrieve a single food item by its food_id.
// It returns a JSON response with the food item if found, or an error message if not.
func GetFood() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a context with a timeout of 100 seconds to avoid long-running DB operations
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		// Extract the 'food_id' parameter from the URL path
		foodId := c.Param("food_id")

		// Declare a variable to store the result of the MongoDB query
		var food models.Food

		// Attempt to find a document in the 'food' collection with the matching food_id
		err := foodCollection.FindOne(ctx, bson.M{"food_id": foodId}).Decode(&food)

		// Cancel the context to free up resources (done after DB operation completes)
		defer cancel()

		// If an error occurred during the database lookup, return an error response
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "error occurred while fetching the food item",
			})
			return
		}

		// If successful, return the food item as a JSON response with HTTP 200 OK status
		c.JSON(http.StatusOK, food)
	}
}

// CreateFood inserts a new food item into the database
func CreateFood() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Create a context with a timeout of 100 seconds to prevent long-running DB operations
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		// Declare a variable to hold the menu data fetched from MongoDB (used for validation)
		var menu models.Menu

		// Declare a variable to hold the incoming food item data from the client
		var food models.Food

		// Bind the incoming JSON request body to the 'food' struct
		// If there is an error (e.g. bad format), return 400 Bad Request
		if err := c.BindJSON(&food); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate the 'food' struct fields using validator rules defined in the model
		// If validation fails, return 400 Bad Request with validation errors
		validationErr := validate.Struct(food)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}
		// Query the 'menu' collection to ensure the provided Menu_id exists
		// This ensures referential integrity (food must belong to a valid menu)
		err := menuCollection.FindOne(ctx, bson.M{"menu_id": food.Menu_id}).Decode(&menu)
		defer cancel()

		// If menu was not found, return 500 Internal Server Error with appropriate message
		if err != nil {
			msg := fmt.Sprintf("menu was not found")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		// Set creation and update timestamps using RFC3339 format
		food.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		food.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		// Generate a new ObjectID for MongoDB (_id) and assign it
		food.ID = primitive.NewObjectID()

		// Convert the generated ObjectID to a string and assign to 'Food_id' (used in API)
		food.Food_id = food.ID.Hex()

		// Round the food price to 2 decimal places using the toFixed helper function
		var num = toFixed(*food.Price, 2)
		food.Price = &num

		// Insert the validated and completed food item into the MongoDB 'food' collection
		result, insertErr := foodCollection.InsertOne(ctx, food)

		// If insertion fails (e.g., DB error), return 500 with error message
		if insertErr != nil {
			msg := fmt.Sprintf("Food item was not created")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		defer cancel()
		c.JSON(http.StatusOK, result)
	}
}

// round helps round a float to the nearest integer
func round(num float64) int {
	// math.Copysign ensures correct rounding direction based on the sign of the number.
	// e.g., round(2.3) = 2, round(2.7) = 3, round(-2.3) = -2, round(-2.7) = -3
	return int(num + math.Copysign(0.5, num))
}

// toFixed limits a float value to a specific precision
func toFixed(num float64, precision int) float64 {
	// Calculate 10 raised to the power of 'precision'
	output := math.Pow(10, float64(precision))

	// Multiply number to shift decimal, round it, then shift back
	return float64(round(num*output)) / output
}

// UpdateFood modifies an existing food item
func UpdateFood() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var menu models.Menu
		var food models.Food

		foodId := c.Param("food_id")

		if err := c.BindJSON(&food); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var updateObj primitive.D

		if food.Name != nil {
			updateObj = append(updateObj, bson.E{Key: "name", Value: food.Name})
		}
		if food.Price != nil {
			updateObj = append(updateObj, bson.E{Key: "price", Value: food.Price})
		}
		if food.Food_image != nil {
			updateObj = append(updateObj, bson.E{Key: "food_image", Value: food.Food_image})
		}
		if food.Menu_id != nil {
			err := menuCollection.FindOne(ctx, bson.M{"menu_id": food.Menu_id}).Decode(&menu)
			defer cancel()
			if err != nil {
				msg := fmt.Sprintf("message:Menu was not fount")
				c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
				return
			}
			updateObj = append(updateObj, bson.E{Key: "menu", Value: food.Price})
		}

		// Update timestamp
		food.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{Key: "updated_at", Value: food.Updated_at})

		upsert := true
		filter := bson.M{"food_id": foodId}
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		// Update food item
		result, err := foodCollection.UpdateOne(
			ctx,
			filter,
			bson.D{{Key: "$set", Value: updateObj}},
			&opt,
		)

		if err != nil {
			msg := fmt.Sprintf("food item update failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}
