package models

// User schema of the user table
type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	City string `json:"city"`
	Age  int64  `json:"age"`
}
