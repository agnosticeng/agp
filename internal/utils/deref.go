package utils

func Deref[T any](v *T) T {
	var zero T

	if v != nil {
		return *v
	}

	return zero
}

func DerefOr[T any](v *T, _default T) T {
	if v != nil {
		return *v
	}

	return _default
}
