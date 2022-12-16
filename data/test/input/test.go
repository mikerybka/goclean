package test

import "strings"

// Hi
const (
	// Test comment
	a        = "hi" //nolint
	b        = 12
	C string = "yo"
)

const D = "yo"

// Test comment
// More
// /
// And more
var Yooo = strings.Builder{} // another

// Hi
type sup struct {
	// Test comment
	// More
	Time string //nolint
} //nolint
