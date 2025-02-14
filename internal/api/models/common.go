package models

// A ID parameter model. Mainly used for Swagger spec generation.
//
// This is used for operations that want the ID of an object in the path
// swagger:parameters cancelSubscription
type Id struct {
	// The ID of the object
	//
	// in: path
	// required: true
	ID string `json:"id"`
}
