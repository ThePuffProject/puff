package puff

import (
	"fmt"     // For panic message
	"strings" // For splitPathParameter
)

// "fmt" // Removed unused import - this line is redundant now

type nodeType int8

const (
	// nodePrefix denotes a 'normal' node. e.g 'api' in /api/users
	nodePrefix nodeType = iota

	// nodePathParam denotes a node representating a path param. e.g 'id' in /api/users/:id
	nodePathParam

	// nodeAny represents wildcard '*'
	nodeAny
)

type node struct {
	// prefix is the value of the node
	prefix string
	routes map[string]*Route
	// allMethods only needed, to avoid looping over methodRoutes to return Allow header
	allMethods []string
	// direct ascendant of node
	parent *node
	legacyChildren []*node // Renamed from children
	staticChildren map[string]*node
	paramChild     *node
	anyChild       *node
	// param    string // what is param even doing?? // Commented out as per previous subtask, seems unused
	type_ nodeType
}

func newNode(prefix string, parent *node) *node {
	return &node{
		prefix:         prefix,
		routes:         map[string]*Route{},
		allMethods:     []string{},
		parent:         parent,
		legacyChildren: []*node{}, // Initialize legacyChildren
		staticChildren: make(map[string]*node),
		paramChild:     nil,
		anyChild:       nil,
		type_:          determineNodeType(prefix),
	}
}

// func insertNode(p string) *node {
// 	segments := segmentPath(p)
// 	if len(segments) == 0 {
// 		return nil // Handle edge cases where the path is empty
// 	}
// 	// Create the root node with the first segment
// 	mountNode := newNode(segments[0], nil)

// 	current := mountNode

// 	// Add children for subsequent segments
// 	for _, segment := range segments[1:] {
// 		child := current.addChild(segment)
// 		current = child
// 	}

// 	return mountNode // Return the root of the hierarchy
// }

// func (n *node) findChild(segment string, nodeType nodeType) *node {
// 	for _, child := range n.legacyChildren { // Updated to legacyChildren
// 		if child.prefix == segment && child.type_ == nodeType {
// 			return child
// 		}
// 	}
// 	return nil
// }

func (n *node) isMethodTaken(method, path string) bool {
	r, exists := n.routes[method]
	if exists {
		return r.Path == path
	}
	return false
}

// func (n *node) addChild(prefix string) *node {
// 	// Validate the prefix
// 	// if prefix == "" {
// 	// 	err := fmt.Errorf("prefix was empty when adding child to node %s", n.prefix)
// 	// 	panic(err)
// 	// }

// 	// Check for duplicate prefixes among children
// 	for _, child := range n.legacyChildren { // Updated to legacyChildren
// 		if child.prefix == prefix {
// 			panic(fmt.Sprintf("child with prefix '%s' already exists under parent '%s'", prefix, n.prefix))
// 		}
// 	}

// 	// Create the new child node
// 	newNode := newNode(prefix, nil)

// 	n.legacyChildren = append(n.legacyChildren, newNode) // Updated to legacyChildren
// 	return newNode
// }

// utils for working with node
func isParam(s string) bool { // Renamed prefix to s to avoid confusion with node.prefix
	if len(s) == 0 {
		return false
	}
	// Path parameters must be at least 3 characters long, e.g., {i}
	if len(s) < 3 {
		return false
	}
	return s[0] == '{' && s[len(s)-1] == '}'
}

func determineNodeType(prefix string) nodeType {
	if len(prefix) > 0 && prefix[0] == '*' {
		// Wildcard nodes must be at least 1 character long, e.g., *
		// And if longer, the second character cannot be '{' to avoid confusion with param nodes like *{param}
		// Though this case might be handled by routing logic itself, adding a check here for robustness.
		// For now, a simple check for '*' is enough as per current requirements.
		return nodeAny
	}
	// Changed to call the updated isParam function
	if isParam(prefix) {
		return nodePathParam
	}
	return nodePrefix
}

// splitPathParameter takes a path starting with a parameter (e.g., "{id}/foo" or "{id}")
// and returns the parameter part (e.g., "{id}") and the rest of the path (e.g., "/foo" or "").
// It assumes the input path string starts with '{'.
func splitPathParameter(fullParamPath string) (paramSegment string, remainingPath string) {
	// This function relies on determineNodeType having already confirmed it's a param path.
	// It also assumes path is not empty and starts with '{'.
	endIndex := strings.Index(fullParamPath, "}")
	if endIndex == -1 {
		// Malformed parameter (e.g., "{id/foo" or just "{id" without closing brace)
		// This case should ideally be caught earlier or implies an invalid path format.
		// For robustness, return the original path as segment, expecting further logic to handle/error.
		return fullParamPath, ""
	}
	paramSegment = fullParamPath[:endIndex+1]

	if endIndex+1 >= len(fullParamPath) {
		remainingPath = ""
	} else {
		remainingPath = fullParamPath[endIndex+1:]
		// Ensure remainingPath, if not empty, starts with a "/" or is handled appropriately by insert.
		// Current insert logic expects segments to be handled by LCP.
		// If remainingPath is "foo", insert will treat it as a static segment.
		// If it's "/foo", LCP with a "/" prefix node would consume the slash.
	}
	return
}


// addMethods appends new HTTP methods to the node's allMethods slice, ensuring no duplicates.
func (n *node) addMethods(newMethods ...string) {
    for _, newMethod := range newMethods {
        exists := false
        for _, existingMethod := range n.allMethods {
            if newMethod == existingMethod {
                exists = true
                break
            }
        }
        if !exists {
            n.allMethods = append(n.allMethods, newMethod)
        }
    }
}

func (n *node) insert(path string, routeToAdd *Route, httpMethods []string) {
    // 1. Handle route addition for empty path (base case for recursion)
    if path == "" {
        if n.routes == nil {
            n.routes = make(map[string]*Route)
        }
        for _, method := range httpMethods {
            n.routes[method] = routeToAdd // Overwrites if method already exists
        }
        n.addMethods(httpMethods...) // Ensures allMethods is updated without duplicates
        return
    }

    // 2. Handle current node 'n' based on its type (for non-empty 'path')
    //    (Path is guaranteed non-empty at this point)
    if n.type_ == nodePathParam || n.type_ == nodeAny {
        // For parameter or wildcard nodes, their own prefix (e.g., "{id}", "*") 
        // is a placeholder that was conceptually matched by the parent.
        // The current 'path' is what remains *after* this conceptual match.
        // We directly proceed to insert this 'path' into the children of n.
        // No LCP/splitting of n's own placeholder prefix is done here.
    } else { // Static node (nodePrefix)
        lcp := longestCommonPrefix(path, n.prefix)

        if lcp < len(n.prefix) {
            // Path diverges from n's static prefix, so node 'n' must be split.
            splitChild := newNode(n.prefix[lcp:], n) // New child gets the divergent part of n's prefix
            splitChild.routes = n.routes
            splitChild.allMethods = n.allMethods
            splitChild.staticChildren = n.staticChildren
            splitChild.paramChild = n.paramChild
            splitChild.anyChild = n.anyChild
            // splitChild.type_ = n.type_ // This was in the prompt, but n.type_ is nodePrefix. newNode determines type.
                                        // The original type of the segment that becomes splitChild is nodePrefix.
                                        // So newNode(n.prefix[lcp:], n) will correctly set its type if it's static.
                                        // If n.prefix[lcp:] by chance looked like a param (e.g. "{foo}"),
                                        // determineNodeType would set it. But this should not happen for a static node split.
                                        // Thus, relying on newNode to set type_ based on the new prefix is fine.
                                        // If the original n had a specific type that needs to be preserved for the child part,
                                        // then splitChild.type_ = n.type_ (before n.type_ is changed) would be needed.
                                        // But here n is nodePrefix, so its child part is also nodePrefix.

            // Update parent pointers for all of splitChild's children to point to splitChild
            for _, childNode := range splitChild.staticChildren {
                childNode.parent = splitChild
            }
            if splitChild.paramChild != nil {
                splitChild.paramChild.parent = splitChild
            }
            if splitChild.anyChild != nil {
                splitChild.anyChild.parent = splitChild
            }
            
            // Update current node 'n' to become the common ancestor.
            n.prefix = n.prefix[:lcp]
            n.routes = make(map[string]*Route) // Clear routes from n, they moved to splitChild
            n.allMethods = []string{}          // Clear methods from n
            n.staticChildren = make(map[string]*node) // Reset children for n
            // Add splitChild as the first child of the (now shorter) n.
            // Key correctly based on the first char of splitChild's new prefix.
            if len(splitChild.prefix) > 0 { // Ensure prefix is not empty before keying
                 n.staticChildren[splitChild.prefix[0:1]] = splitChild 
            }
            n.paramChild = nil                 // Clear from n
            n.anyChild = nil                   // Clear from n
            n.type_ = nodePrefix // Parent of a split is always a static prefix node
        }

        // After potential split, 'n' represents the common part (up to lcp).
        // Consume the LCP from 'path' as it's now represented by 'n'.
        path = path[lcp:]

        // If path is now empty after consuming LCP, the route ends at 'n'.
        if path == "" {
            if n.routes == nil {
                n.routes = make(map[string]*Route)
            }
            for _, method := range httpMethods {
                n.routes[method] = routeToAdd
            }
            n.addMethods(httpMethods...)
            return
        }
        // If path is still not empty, it needs to be inserted into children of 'n'.
    }

    // 3. Common Child Insertion Logic (path is guaranteed non-empty here)
    //    'path' is the segment to be inserted under the current node 'n'.
    childNodeType := determineNodeType(path)

    switch childNodeType {
    case nodePathParam:
        paramSegment, remainingPathAfterParam := splitPathParameter(path) // e.g., "{id}", "/foo"
        if n.paramChild == nil {
            n.paramChild = newNode(paramSegment, n) // paramChild prefix is just "{id}"
            n.paramChild.type_ = nodePathParam
        } else if n.paramChild.prefix != paramSegment {
            panic(fmt.Sprintf("conflicting path parameter definition: existing %s, new %s on parent %s", n.paramChild.prefix, paramSegment, n.prefix))
        }
        n.paramChild.insert(remainingPathAfterParam, routeToAdd, httpMethods)

    case nodeAny:
        // Simplified wildcard handling for now (will need its own splitPathWildcard helper later)
        // This assumes the 'path' is the wildcard segment itself, e.g., "*filepath" or "*"
        // For this subtask, we'll assume the path passed here is the full wildcard segment.
        anySegment := path // e.g., path is "*foo" or "*"
        remainingPathAfterAny := "" // Assume wildcard consumes all of this 'path' segment

        // A more robust version would be:
        // anySegment, remainingPathAfterAny := splitPathWildcard(path)

        if n.anyChild == nil {
            n.anyChild = newNode(anySegment, n) // anyChild prefix is "*foo" or "*"
            n.anyChild.type_ = nodeAny
        } else if n.anyChild.prefix != anySegment {
             panic(fmt.Sprintf("conflicting wildcard definition: existing %s, new %s on parent %s", n.anyChild.prefix, anySegment, n.prefix))
        }
        // Route is for this wildcard node, consuming its prefix.
        n.anyChild.insert(remainingPathAfterAny, routeToAdd, httpMethods)


    case nodePrefix: // Static child
        if n.staticChildren == nil {
            n.staticChildren = make(map[string]*node)
        }
        
        // 'path' is the segment to insert, e.g., "foo/bar" or "foo".
        // This will become the prefix of the new child node if it doesn't conflict/merge.
        // The key for staticChildren map is the first character of this path.
        childKey := path[0:1] 
        // TODO: Consider paths like "/" if they can reach here.
        // If path is "/segment", key currently would be '/'. Child prefix "/segment".
        // If path is "segment", key 's'. Child prefix "segment".
        // This implies prefixes can sometimes start with '/'. Needs consistent handling.
        // For now, assume 'path' here is a relative segment like "foo" or "foo/bar".

        child, exists := n.staticChildren[childKey]
        if !exists {
            child = newNode(path, n) // Child node gets the (potentially multi-segment) path as its prefix
            n.staticChildren[childKey] = child
            // Route is for this new child, meaning it consumes its entire new prefix.
            child.insert("", routeToAdd, httpMethods) 
        } else {
            // Existing child, recurse. child.insert will handle LCP with child.prefix.
            child.insert(path, routeToAdd, httpMethods)
        }
    }
}
