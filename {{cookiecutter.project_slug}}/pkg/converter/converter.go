// Package converter wraps github.com/jinzhu/copier with generic helpers that
// copy structs and slices between types, returning pointers to the result.
package converter

import (
	"github.com/jinzhu/copier"
)

// Convert copies from into a new value of type T.
func Convert[T any](from any) (*T, error) {
	var t T
	if err := copier.Copy(&t, from); err != nil {
		return nil, err
	}
	return &t, nil
}

// ConvertList copies every element of from into a new slice of T. A nil input
// yields a nil slice.
func ConvertList[F any, T any](from *[]F) (*[]T, error) {
	if from == nil {
		return nil, nil
	}
	tList := make([]T, 0, len(*from))
	for _, elem := range *from {
		var t T
		if err := copier.Copy(&t, elem); err != nil {
			return nil, err
		}
		tList = append(tList, t)
	}
	return &tList, nil
}

// ConvertToListPointer copies every element of from into a new slice of *T. A
// nil input yields a nil slice.
func ConvertToListPointer[F any, T any](from *[]F) (*[]*T, error) {
	if from == nil {
		return nil, nil
	}
	tList := make([]*T, 0, len(*from))
	for _, elem := range *from {
		var t T
		if err := copier.Copy(&t, elem); err != nil {
			return nil, err
		}
		tList = append(tList, &t)
	}
	return &tList, nil
}

// MustConvert is [Convert] but panics on error instead of returning it.
func MustConvert[T any](from any) *T {
	t, err := Convert[T](from)
	if err != nil {
		panic(err)
	}
	return t
}

// MustConvertList is [ConvertList] but panics on error instead of returning
// it.
func MustConvertList[F any, T any](from *[]F) *[]T {
	t, err := ConvertList[F, T](from)
	if err != nil {
		panic(err)
	}
	return t
}

// MustConvertToListPointer is [ConvertToListPointer] but panics on error
// instead of returning it.
func MustConvertToListPointer[F any, T any](from *[]F) *[]*T {
	t, err := ConvertToListPointer[F, T](from)
	if err != nil {
		panic(err)
	}
	return t
}
