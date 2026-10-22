package resource

import (
	"testing"
)

func TestOptionalItem(t *testing.T) {
	calls := 0
	transform := func(value int) int { calls++; return value * 2 }
	if Item[int](nil, transform) != nil || calls != 0 {
		t.Fatal("nil must not invoke converter")
	}
	value := 3
	if got := Item(&value, transform); got == nil || *got != 6 || value != 3 {
		t.Fatal("incorrect optional mapping")
	}
}

func TestPageMapsItemsInOrder(t *testing.T) {
	input := Page[int]{Items: []int{3, 1}, Total: 12, Page: 2, PageSize: 2}
	out := Paginate(input, func(value int) int { return value * 2 })
	if out.Total != 12 || out.Page != 2 || out.PageSize != 2 || len(out.Items) != 2 || out.Items[0] != 6 || out.Items[1] != 2 {
		t.Fatalf("unexpected page: %+v", out)
	}
	out.Items[0] = 0
	if input.Items[0] != 3 {
		t.Fatal("resource mutated input")
	}
}
