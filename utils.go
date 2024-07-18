package puff

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"mime"
	"net/http"
	"strings"
)

// RandomNanoID generates a random NanoID with format
// LLLL-NNNN. IMPORTANT: THIS FUNCTION IS NOT
// CRYPTOGRAPHICALLY SECURE. DO NOT USE THIS TO GENERATE
// TOKENS WITH AUTHORITY (instead see RandomToken).
func RandomNanoID() string {
	id := ""
	for range 4 {
		r := rand.IntN(25) + 1
		id += fmt.Sprintf("%c", ('A' - 1 + r))
	}
	id += "-"
	for range 4 {
		r := rand.IntN(9)
		id += fmt.Sprint(r)
	}
	return id
}

// RandomToken generates a crytographically secure
// random base64 token with the provided length.
func RandomToken(length int) string {
	randomBytes := make([]byte, length)
	_, err := cryptorand.Read(randomBytes)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(randomBytes)
}

func resolveContentType(provided, default_content_type string) string {
	if provided == "" {
		return default_content_type
	}
	return provided
}

func resolveStatusCode(provided int, _default int) int {
	if provided == 0 {
		return _default
	}
	return provided
}

func contentTypeFromFileName(name string) string {
	fileNameSplit := strings.Split(name, ".")
	suffix := fileNameSplit[len(fileNameSplit)-1]
	ct := mime.TypeByExtension("." + suffix)
	if ct == "" {
		return "text/plain" // default content type
	}
	return ct
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"error": message})
	w.WriteHeader(statusCode)
}
