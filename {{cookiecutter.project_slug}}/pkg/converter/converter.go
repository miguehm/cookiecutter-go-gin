package converter

import (
	"github.com/jinzhu/copier"
)

func Convert[T any](from any) (*T, error) {
	var t T
	if err := copier.Copy(&t, from); err != nil {
		return nil, err
	}
	return &t, nil
}

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

func MustConvert[T any](from any) *T {
	t, err := Convert[T](from)
	if err != nil {
		panic(err)
	}
	return t
}

func MustConvertList[F any, T any](from *[]F) *[]T {
	t, err := ConvertList[F, T](from)
	if err != nil {
		panic(err)
	}
	return t
}

func MustConvertToListPointer[F any, T any](from *[]F) *[]*T {
	t, err := ConvertToListPointer[F, T](from)
	if err != nil {
		panic(err)
	}
	return t
}
