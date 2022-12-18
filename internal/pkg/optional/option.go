package optional

type Option[T any] interface {
	IsSome() bool
	IsNone() bool
	Unwrap() T
}

type some[T any] struct {
	value T
}

func (s some[T]) IsSome() bool {
	return true
}

func (s some[T]) IsNone() bool {
	return false
}

func (s some[T]) Unwrap() T {
	return s.value
}

var _ Option[int] = some[int]{}

func Some[T any](v T) Option[T] {
	return some[T]{value: v}
}

type none[T any] struct{}

func (s none[T]) IsSome() bool {
	return false
}

func (s none[T]) IsNone() bool {
	return true
}

func (s none[T]) Unwrap() T {
	panic("can not unwrap None")
}

var _ Option[int] = none[int]{}

func None[T any]() Option[T] {
	return none[T]{}
}
