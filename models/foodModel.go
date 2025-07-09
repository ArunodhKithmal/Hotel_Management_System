package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Food represents a food item in the restaurant's menu.
// It includes fields for name, price, image URL, and references to related entities (like menu).
type Food struct {
	ID         primitive.ObjectID `bson:"_id"`                                    // MongoDB document ID
	Name       *string            `json:"name" validate:"required,min=2,max=100"` // Name of the food item (min 2, max 100 characters)
	Price      *float64           `json:"price" validate:"required"`              // Price of the food item (required)
	Food_image *string            `json:"food_image" validate:"required"`         // URL or reference to the image of the food (required)
	Created_at time.Time          `json:"created_at"`                             // Timestamp when the item was created
	Updated_at time.Time          `json:"updated_at"`                             // Timestamp when the item was last updated
	Food_id    string             `json:"food_id"`                                // Human-readable unique ID for the food item
	Menu_id    *string            `json:"menu_id" validate:"required"`            // Reference to the menu this food item belongs to
}
