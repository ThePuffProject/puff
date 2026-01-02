package puff

import (
	"encoding/json"
	"net/http"
)

// common handlers returned inside puff
func ErrMethodNotAllowed(c *Context) {
	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.ResponseWriter.WriteHeader(http.StatusMethodNotAllowed)
	response := map[string]string{
		"error":   "Method Not Allowed for the requested resource.",
		"message": "Method Not Allowed for the requested resource.",
	}
	json.NewEncoder(c.ResponseWriter).Encode(response)
}

// TODO: make the keys passible inside Puff Router as a struct like the key for the error 'ErrorKey' as well as the ErrResponseType which for now can be string or JSON.
// TODO: may need to make it so that maxNumberofConnections can be pulled from Server or something and used as stuff inside context
