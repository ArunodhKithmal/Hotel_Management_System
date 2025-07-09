package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DBinstance establishes a new MongoDB client connection to the localhost server.
// It sets a 10-second timeout for connecting and logs any failure.
func DBinstance() *mongo.Client {
	MongoDb := "mongodb://localhost:27017" // MongoDB URI (local instance)
	fmt.Print(MongoDb)

	// Create a new client using the URI
	client, err := mongo.NewClient(options.Client().ApplyURI(MongoDb))
	if err != nil {
		log.Fatal(err) // Exit the program if client creation fails
	}

	// Create a context with a timeout for the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt to connect to MongoDB
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err) // Exit if connection fails
	}

	fmt.Println("connected to mongodb")
	return client // Return the connected client
}

// Client is a global MongoDB client instance used throughout the project.
var Client *mongo.Client = DBinstance()

// OpenCollection returns a reference to a collection in the "restaurant" database.
// This is used in controller functions to interact with MongoDB collections.
func OpenCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	var Collection *mongo.Collection = client.Database("restaurant").Collection(collectionName)
	return Collection
}
