package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Order represents a customer's order in the system
type Order struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	Order_id   string             `json:"order_id" bson:"order_id"`
	Name       string             `json:"name" bson:"name" validate:"required"`
	Category   string             `json:"category" bson:"category" validate:"required"`
	Start_Date *time.Time         `json:"start_date" bson:"start_date,omitempty"`
	End_Date   *time.Time         `json:"end_date" bson:"end_date,omitempty"`
	Created_at time.Time          `json:"created_at" bson:"created_at"`
	Updated_at time.Time          `json:"updated_at" bson:"updated_at"`
	Menu_id    string             `json:"menu_id,omitempty" bson:"menu_id,omitempty"`
	Table_id   *string            `json:"table_id,omitempty" bson:"table_id,omitempty"`
}
