package exo

import "github.com/gofiber/fiber/v2"

// Get declares a struct that represents a GET request.
type Get struct {
	*fiber.Ctx
}

// Post declares a struct that represents a POST request.
type Post struct {
	*fiber.Ctx
}

// Put declares a struct that represents a PUT request.
type Put struct {
	*fiber.Ctx
}

// Delete declares a struct that represents a DELETE request.
type Delete struct {
	*fiber.Ctx
}

// Patch declares a struct that represents a PATCH request.
type Patch struct {
	*fiber.Ctx
}

// Options declares a struct that represents an OPTIONS request.
type Options struct {
	*fiber.Ctx
}

// Head declares a struct that represents a HEAD request.
type Head struct {
	*fiber.Ctx
}

// Trace declares a struct that represents a TRACE request.
type Trace struct {
	*fiber.Ctx
}

// O is a type that represents a JSON object.
type O map[string]interface{}

// A is a type that represents a JSON array.
type A []interface{}
