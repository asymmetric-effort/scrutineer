package assertion

import (
	"fmt"
	"reflect"
)

// toSlice converts an any value to []any if it is a slice or array.
func toSlice(actual any) ([]any, error) {
	if actual == nil {
		return nil, fmt.Errorf("expected slice or array, got nil")
	}
	v := reflect.ValueOf(actual)
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		result := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			result[i] = v.Index(i).Interface()
		}
		return result, nil
	default:
		return nil, fmt.Errorf("expected slice or array, got %T", actual)
	}
}

// LengthAssertion checks that the length of a slice, array, map, or string
// equals the expected length.
type LengthAssertion struct {
	Expected int
}

// Name returns the assertion name.
func (a *LengthAssertion) Name() string { return "length" }

// Evaluate checks that the actual value has the expected length.
func (a *LengthAssertion) Evaluate(actual any) error {
	if actual == nil {
		if a.Expected == 0 {
			return nil
		}
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Expected,
			Actual:    nil,
			Message:   fmt.Sprintf("expected length %d, got nil", a.Expected),
		}
	}
	v := reflect.ValueOf(actual)
	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String, reflect.Chan:
		length := v.Len()
		if length == a.Expected {
			return nil
		}
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Expected,
			Actual:    length,
			Message:   fmt.Sprintf("expected length %d, got %d", a.Expected, length),
		}
	default:
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Expected,
			Actual:    actual,
			Message:   fmt.Sprintf("cannot get length of %T", actual),
		}
	}
}

// EmptyAssertion checks that a slice, array, map, or string is empty.
type EmptyAssertion struct{}

// Name returns the assertion name.
func (a *EmptyAssertion) Name() string { return "empty" }

// Evaluate checks that the actual value is empty.
func (a *EmptyAssertion) Evaluate(actual any) error {
	if actual == nil {
		return nil
	}
	v := reflect.ValueOf(actual)
	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String, reflect.Chan:
		if v.Len() == 0 {
			return nil
		}
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  "empty",
			Actual:    v.Len(),
			Message:   fmt.Sprintf("expected empty %s, got length %d", v.Kind(), v.Len()),
		}
	default:
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  "empty",
			Actual:    actual,
			Message:   fmt.Sprintf("cannot check emptiness of %T", actual),
		}
	}
}

// CollectionNotEmptyAssertion checks that a slice, array, or map is not empty.
type CollectionNotEmptyAssertion struct{}

// Name returns the assertion name.
func (a *CollectionNotEmptyAssertion) Name() string { return "collection_not_empty" }

// Evaluate checks that the actual value is not empty.
func (a *CollectionNotEmptyAssertion) Evaluate(actual any) error {
	if actual == nil {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  "non-empty collection",
			Actual:    nil,
			Message:   "expected non-empty collection, got nil",
		}
	}
	v := reflect.ValueOf(actual)
	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String, reflect.Chan:
		if v.Len() > 0 {
			return nil
		}
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  "non-empty collection",
			Actual:    v.Len(),
			Message:   fmt.Sprintf("expected non-empty %s", v.Kind()),
		}
	default:
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  "non-empty collection",
			Actual:    actual,
			Message:   fmt.Sprintf("cannot check emptiness of %T", actual),
		}
	}
}

// EachAssertion checks that every element of a slice/array passes the inner assertion.
type EachAssertion struct {
	Inner Assertion
}

// Name returns the assertion name.
func (a *EachAssertion) Name() string { return "each" }

// Evaluate checks that every element passes the inner assertion.
func (a *EachAssertion) Evaluate(actual any) error {
	items, err := toSlice(actual)
	if err != nil {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  "iterable",
			Actual:    actual,
			Message:   err.Error(),
		}
	}
	for i, item := range items {
		if evalErr := a.Inner.Evaluate(item); evalErr != nil {
			return &AssertionError{
				Assertion: a.Name(),
				Expected:  fmt.Sprintf("all elements to pass %s", a.Inner.Name()),
				Actual:    item,
				Message:   fmt.Sprintf("element [%d] failed: %s", i, evalErr.Error()),
			}
		}
	}
	return nil
}

// AnyAssertion checks that at least one element of a slice/array passes the inner assertion.
type AnyAssertion struct {
	Inner Assertion
}

// Name returns the assertion name.
func (a *AnyAssertion) Name() string { return "any" }

// Evaluate checks that at least one element passes the inner assertion.
func (a *AnyAssertion) Evaluate(actual any) error {
	items, err := toSlice(actual)
	if err != nil {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  "iterable",
			Actual:    actual,
			Message:   err.Error(),
		}
	}
	for _, item := range items {
		if evalErr := a.Inner.Evaluate(item); evalErr == nil {
			return nil
		}
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  fmt.Sprintf("at least one element to pass %s", a.Inner.Name()),
		Actual:    actual,
		Message:   "no element passed the assertion",
	}
}

// AllAssertion is an alias for EachAssertion - checks that all elements pass.
type AllAssertion struct {
	Inner Assertion
}

// Name returns the assertion name.
func (a *AllAssertion) Name() string { return "all" }

// Evaluate checks that all elements pass the inner assertion.
func (a *AllAssertion) Evaluate(actual any) error {
	items, err := toSlice(actual)
	if err != nil {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  "iterable",
			Actual:    actual,
			Message:   err.Error(),
		}
	}
	for i, item := range items {
		if evalErr := a.Inner.Evaluate(item); evalErr != nil {
			return &AssertionError{
				Assertion: a.Name(),
				Expected:  fmt.Sprintf("all elements to pass %s", a.Inner.Name()),
				Actual:    item,
				Message:   fmt.Sprintf("element [%d] failed: %s", i, evalErr.Error()),
			}
		}
	}
	return nil
}
