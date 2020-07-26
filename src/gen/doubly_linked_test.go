package gen

import (
	"testing"
)

func TestBasics(t *testing.T) {
	g := NewStringishDoublyLinkedList()
	basicHelper(t, &g)
}
func TestBasicsWithAlloc(t *testing.T) {
	g := NewStringishDoublyLinkedListWithAllocator(nil, nil)
	basicHelper(t, &g)
}

func basicHelper(t *testing.T, g *StringishDoublyLinkedList) {
	t.Helper()

	if !g.Empty() {
		t.Errorf("doubly linked list not empty at start")
	}
	if 0 != g.Length() {
		t.Errorf("doubly linked list not empty at start")
	}
	if g.First() != nil {
		t.Errorf("doubly linked list not empty at start")
	}
	if g.Last() != nil {
		t.Errorf("doubly linked list not empty at start")

	}

	s1 := NewStringish("iansmith")
	s2 := NewStringish("love will tear us apart")
	s3 := NewStringish("the four ladies")

	g.Append(s1)
	if g.Empty() {
		t.Errorf("doubly linked list failed empty test after append")
	}
	if 1 != g.Length() {
		t.Errorf("doubly linked list failed to update length() correct")
	}
	if s1 != g.First().Value() {
		t.Errorf("doubly linked list First() error")
	}
	if s1 != g.Last().Value() {
		t.Errorf("doubly linked list Last() error")
	}

	g.Append(s2)
	if 2 != g.Length() {
		t.Errorf("doubly linked list failed to update length() after 2nd append")
	}
	if s1 != g.First().Value() {
		t.Errorf("doubly linked llist failed to update First() properly")
	}
	if s2 != g.Last().Value() {
		t.Errorf("doubly linked llist failed to update Last() properly")
	}
	if s2 != g.First().Next().Value() {
		t.Errorf("doubly linked llist failed to update Last() properly")
	}

	g.Push(s3)
	if 3 != g.Length() {
		t.Errorf("doubly linked list failed to update Length() after 3rd (push)")
	}
	if s3 != g.First().Value() {
		t.Errorf("doubly linked list failed to update First() correctly after 3rd (push)")
	}
	if s1 != g.First().Next().Value() {
		t.Errorf("doubly linked list failed to update First().Next() correctly after 3rd (push)")
	}
	if s3 != g.Last().Prev().Prev().Value() {
		t.Errorf("doubly linked list failed to update First().Last().Prev().Prev() correctly after 3rd (push)")
	}
	if s2 != g.Last().Value() {
		t.Errorf("doubly linked list failed to update Last() correctly after 3rd (push)")
	}
	if s2 != g.Last().Prev().Value() {
		t.Errorf("doubly linked list failed to update Last().Prev() correctly after 3rd (push)")
	}
	if nil != g.Last().Next() {
		t.Errorf("doubly linked list last is not last!")
	}
	if nil != g.First().Prev() {
		t.Errorf("doubly linked list first is not first!")
	}

	total := len(s1.S) + len(s2.S) + len(s3.S)
	count := 0
	g.TraverseStringish(func(v *Stringish) error {
		count += v.L
		return nil
	})
	if total != count {
		t.Errorf("doubly linked list traversal test")
	}
}
