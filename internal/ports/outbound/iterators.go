package outbound

import "context"

// Iterator provides sequential access to a stream of values.
type Iterator[T any] interface {
	Next(ctx context.Context) (T, bool, error)
	Close() error
}

// SliceIterator iterates over a slice of values.
type SliceIterator[T any] struct {
	values []T
	index  int
}

// NewSliceIterator constructs an iterator for the provided slice.
func NewSliceIterator[T any](values []T) *SliceIterator[T] {
	return &SliceIterator[T]{values: values}
}

// Next returns the next value in the iterator.
func (it *SliceIterator[T]) Next(ctx context.Context) (T, bool, error) {
	var zero T
	if err := ctx.Err(); err != nil {
		return zero, false, err
	}
	if it.index >= len(it.values) {
		return zero, false, nil
	}
	value := it.values[it.index]
	it.index++
	return value, true, nil
}

// Close releases resources held by the iterator.
func (it *SliceIterator[T]) Close() error {
	return nil
}

// Collect drains an iterator into a slice.
func Collect[T any](ctx context.Context, it Iterator[T]) ([]T, error) {
	defer it.Close()
	values := []T{}
	for {
		value, ok, err := it.Next(ctx)
		if err != nil {
			return nil, err
		}
		if !ok {
			return values, nil
		}
		values = append(values, value)
	}
}
