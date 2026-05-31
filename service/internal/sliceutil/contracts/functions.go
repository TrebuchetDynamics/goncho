package contracts

// Predicate reports whether value should be accepted.
type Predicate[T any] func(T) bool

// Mapper converts one value into another value.
type Mapper[T any, U any] func(T) U

// FilterMapper converts one value into another value and reports whether to keep it.
type FilterMapper[T any, U any] func(T) (U, bool)

// Keyer derives an accepted comparable key for indexing a value.
type Keyer[T any, K comparable] func(T) (K, bool)
