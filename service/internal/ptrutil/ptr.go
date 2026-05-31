package ptrutil

// Bool returns a pointer to v for optional boolean API fields.
func Bool(v bool) *bool {
	return &v
}
