package resource

import (
	"encoding/json"
	"testing"

	"github.com/xvq/go-template/internal/util/query"
)

func TestOptionalItem(t *testing.T) {
	calls := 0
	transform := func(value int) int { calls++; return value * 2 }
	mapper := New(transform)
	if mapper.Item(nil) != nil || calls != 0 {
		t.Fatal("nil must not invoke converter")
	}
	value := 3
	if got := mapper.Item(&value); got == nil || *got != 6 || value != 3 {
		t.Fatal("incorrect optional mapping")
	}
}

func TestNestedResources(t *testing.T) {
	type source struct {
		Child    *int
		Children []int
	}
	type output struct {
		Child    *int  `json:"child"`
		Children []int `json:"children"`
	}
	child := New(func(value int) int { return value * 2 })
	parent := New(func(value source) output {
		return output{Child: child.Item(value.Child), Children: child.Collection(value.Children)}
	})
	value := 3
	input := query.NewPage([]source{{Child: &value, Children: []int{1, 2}}, {}}, 5, 2, 2)
	page := parent.Page(input)
	if page.Pagination != input.Pagination {
		t.Fatal("nested mapping changed pagination")
	}
	raw, err := json.Marshal(page.Data)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `[{"child":6,"children":[2,4]},{"child":null,"children":[]}]` {
		t.Fatalf("unexpected nested resources: %s", raw)
	}
	if value != 3 || input.Data[0].Children[0] != 1 {
		t.Fatal("nested mapping mutated source")
	}
}

func TestPageMapsItemsInOrder(t *testing.T) {
	input := query.NewPage([]int{3, 1}, 12, 2, 2)
	out := New(func(value int) int { return value * 2 }).Page(input)
	if out.Pagination.Total != 12 || out.Pagination.CurrentPage != 2 || out.Pagination.PerPage != 2 || out.Pagination.LastPage != 6 || out.Pagination.Count != 2 || len(out.Data) != 2 || out.Data[0] != 6 || out.Data[1] != 2 {
		t.Fatalf("unexpected page: %+v", out)
	}
	out.Data[0] = 0
	if input.Data[0] != 3 {
		t.Fatal("resource mutated input")
	}
}

func TestPageJSONPagination(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input query.Page[int]
		want  string
	}{
		{"partial page", query.NewPage([]int{1, 2}, 5, 2, 2), `{"data":[1,2],"pagination":{"current_page":2,"last_page":3,"per_page":2,"total":5,"count":2}}`},
		{"last page", query.NewPage([]int{5}, 5, 3, 2), `{"data":[5],"pagination":{"current_page":3,"last_page":3,"per_page":2,"total":5,"count":1}}`},
		{"empty", query.NewPage[int](nil, 0, 1, 10), `{"data":[],"pagination":{"current_page":1,"last_page":1,"per_page":10,"total":0,"count":0}}`},
		{"out of range", query.NewPage[int](nil, 5, 4, 2), `{"data":[],"pagination":{"current_page":4,"last_page":3,"per_page":2,"total":5,"count":0}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(New(func(value int) int { return value }).Page(tc.input))
			if err != nil {
				t.Fatal(err)
			}
			if string(raw) != tc.want {
				t.Fatalf("unexpected page JSON: %s", raw)
			}
		})
	}
}
