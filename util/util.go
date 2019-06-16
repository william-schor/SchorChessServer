package util

import "fmt"

// Message struct represents a message to and from the client
type Message struct {
	Type       string
	Move       string
	OppoRating string
	OppoName   string
}

// Check handles errors
func Check(err error, message string) {
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", message)
}
