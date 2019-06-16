package util

import (
	"encoding/json"
	"fmt"
	"net"
)

// Message struct represents a message to and from the client
type Message struct {
	Type string
	Val  string
}

// SendMessage sends a json block to the specified client
func SendMessage(conn net.Conn, TypeArg string, ValArg string) {
	m := Message{Type: TypeArg, Val: ValArg}
	b, err := json.Marshal(m)
	Check(err, "JSON written")
	conn.Write(append(b, "\n"...))
}

// Check handles errors
func Check(err error, message string) {
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", message)
}
