package controller

import (
	"context"
	"fmt"
	"golang-Hotel_Management/database"
	"golang-Hotel_Management/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InvoiceViewFormat struct {
	Invoice_id       string      `json:"invoice_id"`
	Payment_method   string      `json:"payment_method"`
	Order_id         string      `json:"order_id"`
	Payment_status   *string     `json:"payment_status"`
	Payment_due      interface{} `json:"payment_due"`
	Table_number     interface{} `json:"table_number"`
	Payment_due_date time.Time   `json:"payment_due_date"`
	Order_details    interface{} `json:"order_details"`
}

var invoiceCollection *mongo.Collection = database.OpenCollection(database.Client, "invoice")

// GetInvoices handles the GET request to retrieve all invoices from the database.
func GetInvoices() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Create a context with a timeout of 100 seconds to avoid long-running operations
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		// Query the 'invoice' collection and retrieve all documents (bson.M{} = match all)
		result, err := invoiceCollection.Find(context.TODO(), bson.M{})

		// Cancel the context to release resources once the DB operation is complete
		defer cancel()

		// If the query fails (e.g., DB connection issue), return a 500 error response
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "error occurred while listing invoice items",
			})
			return
		}

		// Create a slice to store all fetched invoice documents
		var allInvoices []bson.M

		// Decode all documents from the cursor into the allInvoices slice
		if err = result.All(ctx, &allInvoices); err != nil {
			// Log and crash the app if decoding fails (not recommended for production)
			log.Fatal(err)
		}

		// Return a 200 OK response with all invoice documents in JSON format
		c.JSON(http.StatusOK, allInvoices)
	}
}

// GetInvoice handles GET requests to retrieve a specific invoice by invoice_id
func GetInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Set up a context with timeout to avoid long-running DB operations
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		// Extract invoice ID from URL parameter
		invoiceId := c.Param("invoice_id")

		// Declare a variable to store the invoice fetched from the database
		var invoice models.Invoice

		// Query MongoDB to find the invoice by invoice_id
		err := invoiceCollection.FindOne(ctx, bson.M{"invoice_id": invoiceId}).Decode(&invoice)

		// Cancel the context after DB operation completes
		defer cancel()

		// If the invoice is not found or query fails, return an error response
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing invoice item"})
			return
		}

		// Declare a variable to store the formatted invoice view for the response
		var invoiceView InvoiceViewFormat

		// Call helper function to fetch all items associated with the order in this invoice
		allOrderItems, err := ItemsByOrder(invoice.Order_id)

		// If fetching order items fails, return an error response
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while retrieving order items"})
			return
		}

		// Begin building the custom invoice view for client-friendly response

		// Set the order ID in the invoice view
		invoiceView.Order_id = invoice.Order_id

		// Set the payment due date
		invoiceView.Payment_due_date = invoice.Payment_due_date

		// Default payment method is \"null\" string
		invoiceView.Payment_method = "null"

		// If payment method exists (not nil), use its value
		if invoice.Payment_method != nil {
			invoiceView.Payment_method = *invoice.Payment_method
		}

		// Set other invoice fields in the view
		invoiceView.Invoice_id = invoice.Invoice_id
		invoiceView.Payment_status = invoice.Payment_status

		// Set payment due, table number, and order details from the order items slice
		invoiceView.Payment_due = allOrderItems[0]["payment_due"]
		invoiceView.Table_number = allOrderItems[0]["table_number"]
		invoiceView.Order_details = allOrderItems[0]["order_items"]

		// Return the formatted invoice data as JSON with 200 OK status
		c.JSON(http.StatusOK, invoiceView)
	}
}

func CreateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var invoice models.Invoice

		if err := c.BindJSON(&invoice); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var order models.Order
		err := orderCollection.FindOne(ctx, bson.M{"order_id": invoice.Order_id}).Decode(&order)
		defer cancel()
		if err != nil {
			msg := fmt.Sprintf("message: Order was not found")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		status := "PENDING"
		if invoice.Payment_status == nil {
			invoice.Payment_status = &status
		}

		invoice.Payment_due_date, _ = time.Parse(time.RFC3339, time.Now().AddDate(0, 0, 1).Format(time.RFC3339))
		invoice.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		invoice.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		invoice.ID = primitive.NewObjectID()
		invoice.Invoice_id = invoice.ID.Hex()

		validationErr := validate.Struct(invoice)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		result, insertErr := invoiceCollection.InsertOne(ctx, invoice)
		if insertErr != nil {
			msg := fmt.Sprintf("invoice item was not created")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		defer cancel()
		c.JSON(http.StatusOK, result)
	}
}

// UpdateInvoice handles PUT/PATCH requests to update an existing invoice in the database.
func UpdateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Create a context with a 100-second timeout for database operations
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // Ensure resources are released after execution

		// Declare a variable to hold the updated invoice data from the client
		var invoice models.Invoice

		// Extract the invoice ID from the URL path parameter
		invoiceId := c.Param("invoice_id")

		// Attempt to bind the incoming JSON to the invoice struct
		if err := c.BindJSON(&invoice); err != nil {
			// If the JSON is invalid or doesn't match the struct, return 400 Bad Request
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Define the filter to find the invoice document to update
		filter := bson.M{"invoice_id": invoiceId}

		// Create an empty update object (primitive.D is a BSON document)
		var updateObj primitive.D

		// If Payment_method is provided, add it to the update object
		if invoice.Payment_method != nil {
			updateObj = append(updateObj, bson.E{Key: "payment_method", Value: invoice.Payment_method})
		}

		// If Payment_status is provided, add it to the update object
		if invoice.Payment_status != nil {
			updateObj = append(updateObj, bson.E{Key: "payment_status", Value: invoice.Payment_status})
		}

		// Always update the 'updated_at' timestamp to the current time
		invoice.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{Key: "updated_at", Value: invoice.Updated_at})

		// Set up the update options: Upsert = true (create the document if it doesn't exist)
		upsert := true
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		// If Payment_status is missing, default it to "PENDING"
		status := "PENDING"
		if invoice.Payment_status == nil {
			invoice.Payment_status = &status
		}

		// Perform the update operation on the invoice collection
		result, err := invoiceCollection.UpdateOne(
			ctx,                                     // Context
			filter,                                  // Filter to match the invoice
			bson.D{{Key: "$set", Value: updateObj}}, // Update document
			&opt,                                    // Options (with upsert)
		)

		// If the update fails, return a 500 Internal Server Error
		if err != nil {
			msg := fmt.Sprintf("invoice item update failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		// If successful, return the update result (includes modified count, etc.)
		c.JSON(http.StatusOK, result)
	}
}
