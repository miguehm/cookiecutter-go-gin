package converter

import (
	"testing"
)

type source struct {
	Name string
	Age  int
}

type target struct {
	Name string
	Age  int
}

func TestMustConvert(t *testing.T) {
	t.Parallel()

	got := MustConvert[target](source{Name: "alice", Age: 30})
	if got.Name != "alice" || got.Age != 30 {
		t.Fatalf("unexpected conversion: %+v", got)
	}
}

func TestMustConvertList(t *testing.T) {
	t.Parallel()

	alice := source{Name: "alice", Age: 30}
	bob := source{Name: "bob", Age: 40}
	from := []source{alice, bob}
	got := MustConvertList[source, target](&from)
	if got == nil || len(*got) != 2 {
		t.Fatalf("unexpected list: %+v", got)
	}
	if (*got)[1].Name != "bob" || (*got)[1].Age != 40 {
		t.Fatalf("unexpected conversion: %+v", (*got)[1])
	}
}

func TestMustConvertListNil(t *testing.T) {
	t.Parallel()

	if got := MustConvertList[source, target](nil); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestMustConvertToListPointer(t *testing.T) {
	t.Parallel()

	alice := source{Name: "alice", Age: 30}
	from := []source{alice}
	got := MustConvertToListPointer[source, target](&from)
	if got == nil || len(*got) != 1 {
		t.Fatalf("unexpected list: %+v", got)
	}
	if (*got)[0].Name != "alice" || (*got)[0].Age != 30 {
		t.Fatalf("unexpected conversion: %+v", (*got)[0])
	}
}
