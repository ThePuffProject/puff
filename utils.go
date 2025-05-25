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

func resolveContentType(provided, _default string) string {
	if provided == "" {
		return _default
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

func isAnyOfThese[T comparable](value T, these ...T) bool {
	for _, t := range these {
		if t == value {
			return true
		}
	}
	return false
}

func resolveBool(spec string, def bool) (b bool, err error) {
	switch spec {
	case "":
		b = def
	case "true":
		b = true
	case "false":
		b = false
	default:
		b = def
		err = fmt.Errorf("specified boolean on field must be either true or false")
		return
	}
	return
}

func segmentPath(path string) []string {
	// Remove leading and trailing slashes
	path = strings.Trim(path, "/")

	// handle special characters separately

	if i := strings.LastIndex(path, "."); i > 0 && !strings.Contains(path[i:], "/") {
		return append(strings.Split(path[:i], "/"), path[i:])
	}

	return strings.Split(path, "/")
}

func longestCommonPrefix(a, b string) int {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return i
}

// extractParamName extracts the name from a parameter prefix.
// Examples: "{id}" -> "id", "*filepath" -> "filepath", "*" -> "wildcard", "*{name}" -> "name".
func extractParamName(prefix string) string {
	if len(prefix) > 0 && prefix[0] == '{' && prefix[len(prefix)-1] == '}' {
		// Handles "{param}"
		if len(prefix) > 2 { // Ensure there's something between {}
			return prefix[1 : len(prefix)-1]
		}
	}
	if len(prefix) > 0 && prefix[0] == '*' {
		// Handles "*" or "*{name}" or "*name"
		if len(prefix) > 1 {
			if prefix[1] == '{' && len(prefix) > 2 && prefix[len(prefix)-1] == '}' {
				// Handles "*{name}"
				if len(prefix) > 3 { // Ensure *{n}
					return prefix[2 : len(prefix)-1]
				}
				// Case like "*{}" - treat as wildcard or error? For now, wildcard.
			} else {
				// Handles "*name"
				return prefix[1:]
			}
		}
		return "wildcard" // Default for anonymous "*"
	}
	// Should ideally not be reached for validly structured param/any node prefixes.
	// Returning the original prefix might indicate an issue with node prefixing.
	return prefix
}

// joinPaths combines a base path and a segment path, ensuring a single slash between them.
// Handles various edge cases like empty paths or slashes at boundaries.
func joinPaths(base, segment string) string {
	// Normalize base: ensure it's not empty and ends with a slash if it's not just "/"
	if base == "" || base == "/" {
		base = "/"
	} else {
		base = strings.TrimSuffix(base, "/") // remove potential trailing slash from base
	}

	// Normalize segment: remove leading slash
	segment = strings.TrimPrefix(segment, "/")

	// Handle cases where segment might be empty after trimming
	if segment == "" {
		if base == "/" { // if base was also just "/", return "/"
			return "/"
		}
		// if segment is empty, result is just the (trimmed) base
		return base 
	}
	
	// If base was just "/", avoid double slash at the beginning if segment is not empty
	if base == "/" {
		return "/" + segment
	}

	return base + "/" + segment
}
