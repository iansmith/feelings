package gen

import (
	"github.com/cheekybits/genny/generic"
)

type Generic generic.Type

type GenericNodeDL struct {
	prev  *GenericNodeDL
	next  *GenericNodeDL
	value *Generic
}

// GenericDoublyLinkedList implements a doubly linked list
// that is not concurrent safe.
type GenericDoublyLinkedList struct {
	first       *GenericNodeDL
	last        *GenericNodeDL
	allocator   GenericAllocator
	deallocator GenericDeallocator
}

// Next returns the next element of the list.  This is probably only
// needed by people doing specialized traversals that are too complex
// for Traverse and TraverseBackwards.  Returns nil for the last node
// in the list.
func (g *GenericNodeDL) Next() *GenericNodeDL {
	return g.next
}

// Prev returns the previous element of the list.  This is probably only
// needed by people doing specialized traversals that are too complex
// for Traverse and TraverseBackwards.  Returns nil for the first node
// in the list.
func (g *GenericNodeDL) Prev() *GenericNodeDL {
	return g.prev
}

// Value returns the element's value. This is used by traversal
// functions and others because they are given a GenericNodeDL not
// just a Generic.
func (g *GenericNodeDL) Value() *Generic {
	return g.value
}

// NewGenericDoublyLinkedList returns an empty doubly linked list.
// Note: It returns a value, not a pointer but the methods have
// pointer receivers.
func NewGenericDoublyLinkedList() GenericDoublyLinkedList {
	return GenericDoublyLinkedList{first: nil, last: nil, allocator: nil}
}

// NewGenericDoublyLinkedListWithAllocator returns an empty doubly linked list.
// The allocator provided will be used to create nodes in the lists of this type.
// Note: It returns a value, not a pointer but the methods have
// pointer receivers.
type GenericAllocator func() (*GenericNodeDL, *Generic)
type GenericDeallocator func(*GenericNodeDL, *Generic)

func NewGenericDoublyLinkedListWithAllocator(alloc GenericAllocator,
	dealloc GenericDeallocator) GenericDoublyLinkedList {
	return GenericDoublyLinkedList{first: nil, last: nil,
		allocator: alloc, deallocator: dealloc}
}

// Empty returns true if the list is empty.
func (g *GenericDoublyLinkedList) Empty() bool {
	if g.first == nil {
		if g.last != nil {
			panic("invariant violated checking for Empty")
		}
		return true
	}
	return false
}

// Length returns the number of elements in the list.  This
// requires walking the list.
func (g *GenericDoublyLinkedList) Length() int {
	if g.first == nil {
		if g.last != nil {
			panic("invariant violated in Length")
		}
		return 0
	}
	l := 0
	if err := g.TraverseGeneric(func(_ *Generic) error { l++; return nil }); err != nil {
		panic("unable to compute size due to error in traversal:" + err.Error())
	}
	return l
}

// First returns the first node in the list or a nil if the list is empty.
func (g *GenericDoublyLinkedList) First() *GenericNodeDL {
	if g.first == nil {
		if g.last != nil {
			panic("invariant violated getting First()")
		}
		return nil
	}
	if g.first.prev != nil {
		panic("invariant of first node violated (First())")
	}
	return g.first
}

// Last returns the last node in the list or a nil if the list is empty.
func (g *GenericDoublyLinkedList) Last() *GenericNodeDL {
	if g.last == nil {
		if g.first != nil {
			panic("invariant violated getting Last()")
		}
		return nil
	}
	if g.last.next != nil {
		panic("invariant of last node violated (Last())")
	}
	return g.last
}

// Push creates a new node and puts it at the front of the list. Traversals
// that start at the front will see the  newly created node first.  Returns
// a pointer to the newly created value so you can fill in the fields.
func (g *GenericDoublyLinkedList) Push() *Generic {
	if g.allocator != nil {
		node, value := g.allocator()
		g.PushNode(node)
		return value
	}
	v := new(Generic)
	n := &GenericNodeDL{prev: nil, next: nil, value: v}
	g.PushNode(n)
	return n.value
}

// Push inserts the given node at the front of the list.
// Traversals that start at the front will see the newly
// pushed node first.  Returns the newly modified list.
func (g *GenericDoublyLinkedList) PushNode(n *GenericNodeDL) *GenericDoublyLinkedList {

	if g.first == nil {
		if g.last != nil {
			panic("invariant of empty list is broken (push)")
		}
		g.first = n
		g.last = n
		return g
	}
	old := g.first
	if old.prev != nil {
		panic("invariant of first node of list is broken (push)")
	}
	g.first = n
	old.prev = n
	n.next = old
	n.prev = nil
	return g
}

// Append inserts a new node at the end of the list. Traversals that start at
// the front will see the newly created node last.  Returns a pointer to
// the new value so you can fill in the fields.
func (g *GenericDoublyLinkedList) Append() *Generic {
	if g.allocator != nil {
		node, value := g.allocator()
		g.AppendNode(node)
		return value
	}
	value := new(Generic)
	n := &GenericNodeDL{prev: nil, next: nil, value: value}
	g.AppendNode(n)
	return n.value
}

// Append inserts the given value at the end of the list.  Traversals
// that start at the front will see the newly pushed node last.
// Returns the newly modified list.  This method does a check to insure
// that Next() and Prev() of the new node are nil. If they are not nil,
// it panics.
func (g *GenericDoublyLinkedList) AppendNode(n *GenericNodeDL) *GenericDoublyLinkedList {
	if g.last == nil {
		if g.first != nil {
			panic("invariant of empty list is broken (AppendNode)")
		}
		g.first = n
		g.last = n
		return g
	}
	old := g.last
	if old.next != nil {
		panic("invariant of last node of list is broken (AppendNode)")
	}
	if n.Next() != nil || n.Prev() != nil {
		panic("attempt to insert node that is likely a member of " +
			"another list (AppendNode)")
	}
	g.last = n
	old.next = n
	n.prev = old
	n.next = nil
	return g
}

// Traverse walks all the nodes in the list, in order, starting at the
// front.  It is ok to modify elements that are "behind" the current
// node in the iteration.  So modifying current.prev is ok, but modifying
// current.next is not.  If the iteration function returns an error,
// the traversal is halted and that error is returned as the result of
// TraverseNodes function.
func (g *GenericDoublyLinkedList) TraverseNodesGeneric(fn func(v *GenericNodeDL) error) error {
	curr := g.first
	for curr != nil {
		err := fn(curr)
		if err != nil {
			return err
		}
		curr = curr.next
	}
	return nil
}

// Traverse walks all the items in the list, in order, starting at the
// front. This passes the _value_ of each node to the function supplied
// and the nodes in the list cannot be modified during traversal.
// If the iteration function returns an error, the traversal is halted
// and that error is returned as the result of Traverse function.
func (g *GenericDoublyLinkedList) TraverseGeneric(fn func(v *Generic) error) error {
	curr := g.first
	for curr != nil {
		err := fn(curr.value)
		if err != nil {
			return err
		}
		curr = curr.next
	}
	return nil
}

// TraverseBackwards walks all the nodes in the list, in reverse order,
// starting at the last element.  It is ok to modify elements that are
// "in front of" the current node in the iteration.  So modifying
// current.next is ok, but modifying current.prev is not.  If the
// iteration function returns an error, the traversal is halted and
// that error is returned as the result of TraverseNodesBackwards function.
func (g *GenericDoublyLinkedList) TraverseNodesBackwardsGeneric(fn func(v *GenericNodeDL) error) error {
	curr := g.last
	for curr != nil {
		err := fn(curr)
		if err != nil {
			return err
		}
		curr = curr.prev
	}
	return nil
}

// Nth returns the node that is the Nth element of the list, or nil if there are
// insufficient nodes in the list to reach the Nth.
func (g *GenericDoublyLinkedList) Nth(i int) *GenericNodeDL {
	ct := 0
	current := g.First()
	for ct < g.Length() {
		if ct == i {
			return current
		}
		ct++
		current = current.Next()
	}
	return nil
}

// TraverseBackwards walks all the _values_ in the list, in reverse order,
// starting at the last element.  This function does not allow the caller
// to modify the list as the it is traversed.  If the iteration function
// returns an error, the traversal is halted and that error is returned as
// the result of TraverseBackwards function.
func (g *GenericDoublyLinkedList) TraverseBackwardsGeneric(fn func(v *Generic) error) error {
	curr := g.last
	for curr != nil {
		err := fn(curr.value)
		if err != nil {
			return err
		}
		curr = curr.prev
	}
	return nil
}

// Remove takes a node out of the list.
func (g *GenericDoublyLinkedList) Remove(n *GenericNodeDL) {
	if n.prev == nil || n.next == nil {
		// ONLY element
		if g.first == n && g.last == n {
			g.first = nil
			g.last = nil
			n.prev = nil
			n.next = nil
		}
		//two edge cases
		if n.prev == nil && g.first == n {
			g.first = n.next
			n.next.prev = nil
			n.prev = nil
			n.next = nil
		}
		if n.next == nil && g.last == n {
			g.last = n.prev
			n.prev.next = nil
			n.prev = nil
			n.next = nil
		}
		panic("invariant of removing first or last element violated")
	}
	n.prev.next = n.next
	n.next.prev = n.prev
	n.next = nil
	n.prev = nil
}

// RemoveAndRelease takes a node out of the list and returns it and its value
// to the deallocator.  If there is no deallocator, this is the same as Remove().
func (g *GenericDoublyLinkedList) RemoveAndRelease(n *GenericNodeDL) {
	g.Remove(n)
	if g.deallocator != nil {
		g.deallocator(n, n.value)
	}
}

// InsertBefore takes in the node before which to insert the second
// parameter.  It is permitted to give nil as the value of target and this
// makes this function perform AppendNode().
func (g *GenericDoublyLinkedList) InsertBefore(target *GenericNodeDL,
	n *GenericNodeDL) {

	if target == nil {
		g.AppendNode(n)
		return
	}
	prev := target.prev
	if prev == nil {
		if g.first != target {
			panic("invariant violated with first element (InsertBefore)")
		}
		g.first = n
		n.next = target
		target.prev = n
	}
	if prev.next != target {
		panic("invariant violated with intermediate node (InsertBefore)")
	}
	prev.next = n
	n.prev = prev
	target.prev = n
	n.next = target
}

// InsertAfter takes in the node after which to insert the second
// parameter.  It is permitted to give nil as the value of target and this
// makes this function perform PushNode().
func (g *GenericDoublyLinkedList) InsertAfter(target *GenericNodeDL,
	n *GenericNodeDL) {

	if target == nil {
		g.PushNode(n)
		return
	}
	next := target.next
	if next == nil {
		if g.last != target {
			panic("invariant violated with last element (InsertAfter)")
		}
		g.last = n
		target.next = n
		n.prev = target
	}
	if next.prev != target {
		panic("invariant violated with intermediate node (InsertAfter)")
	}
	next.prev = n
	n.next = next
	n.prev = target
	target.next = n
}

// Pop is a shorthand for Remove(First()) and it returns the removed
// node.
func (g *GenericDoublyLinkedList) Pop() *GenericNodeDL {
	f := g.First()
	g.Remove(f)
	return f
}

// PopAndRelease is a shorthand for RemoveAndRelease(First())
func (g *GenericDoublyLinkedList) PopAndRelease() {
	f := g.First()
	g.RemoveAndRelease(f)
}

// Dequeue is a shorthand for Remove(Last()) and it returns the removed
// node.
func (g *GenericDoublyLinkedList) Dequeue() *GenericNodeDL {
	f := g.Last()
	g.Remove(f)
	return f
}

// DequeueAndRelease is a shorthand for RemoveAndRelease(Last())
func (g *GenericDoublyLinkedList) DequeueAndRelease() {
	f := g.Last()
	g.RemoveAndRelease(f)
}
