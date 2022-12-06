package model

// AssertError identifies an error that indicates an internal code consistency
// issue and should be treated as a critical and unrecoverable error.
type AssertError string

// Error returns the assertion error as a huma-readable string and satisfies
// the error interface.
func (e AssertError) Error() string {
	return "assertion failed: " + string(e)
}

// ErrDeserialize signifies that a problem was encountered when deserializing
// data.
type ErrDeserialize string

// Error implements the error interface.
func (e ErrDeserialize) Error() string {
	return string(e)
}

// isDeserializeErr returns whether or not the passed error is an errDeserialize
// error.
func IsDeserializeErr(err error) bool {
	_, ok := err.(ErrDeserialize)
	return ok
}
