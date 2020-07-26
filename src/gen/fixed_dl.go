package gen

type GenericFixedDL struct {
	pool  *GenericManagedPool
	nodes *GenericNodeDLManagedPool
	dl    GenericDoublyLinkedList
}

func NewGenericFixedDL(nodes *GenericNodeDLManagedPool, pool *GenericManagedPool) GenericFixedDL {
	result := GenericFixedDL{
		pool: pool, nodes: nodes,
	}
	dl := NewGenericDoublyLinkedListWithAllocator(result.Alloc, result.Dealloc)
	result.dl = dl
	return result
}

func (g *GenericFixedDL) Alloc() (*GenericNodeDL, *Generic) {
	elem := g.pool.Alloc()
	node := g.nodes.Alloc()
	if elem == nil || node == nil {
		return nil, nil
	}
	node.value = elem
	node.prev = nil
	node.next = nil
	return node, elem
}

func (g *GenericFixedDL) Dealloc(n *GenericNodeDL, e *Generic) {
	n.next = nil
	n.prev = nil
	n.value = nil
	g.pool.Dealloc(e)
	g.nodes.Dealloc(n)
}

func (g *GenericFixedDL) PoolsEmpty() bool {
	return g.nodes.Empty() && g.pool.Empty()
}
func (g *GenericFixedDL) PoolsFull() bool {
	return g.nodes.Full() && g.pool.Full()
}

//
// Wrappers around the DoublyLinkedList functions of the same name.
//
func (g *GenericFixedDL) Append() *Generic {
	return g.dl.Append()
}
func (g *GenericFixedDL) Push() *Generic {
	return g.dl.Push()
}
func (g *GenericFixedDL) Empty() bool {
	return g.dl.Empty()
}
func (g *GenericFixedDL) Length() int {
	return g.dl.Length()
}
func (g *GenericFixedDL) Pop() *GenericNodeDL {
	f := g.dl.First()
	g.dl.Remove(f)
	return f
}
func (g *GenericFixedDL) PopAndRelease() {
	f := g.dl.First()
	g.dl.RemoveAndRelease(f)
}
func (g *GenericFixedDL) Dequeue() *GenericNodeDL {
	f := g.dl.Last()
	g.dl.Remove(f)
	return f
}
func (g *GenericFixedDL) DequeueAndRelease() {
	f := g.dl.Last()
	g.dl.RemoveAndRelease(f)
}
func (g *GenericFixedDL) First() *GenericNodeDL {
	return g.dl.First()
}
func (g *GenericFixedDL) Last() *GenericNodeDL {
	return g.dl.Last()
}
func (g *GenericFixedDL) Remove(n *GenericNodeDL) {
	g.dl.Remove(n)
}
func (g *GenericFixedDL) RemoveAndRelease(n *GenericNodeDL) {
	g.dl.RemoveAndRelease(n)
}
func (g *GenericFixedDL) Nth(i int) *GenericNodeDL {
	return g.dl.Nth(i)
}
func (g *GenericFixedDL) InsertAfter(target *GenericNodeDL, newItem *GenericNodeDL) {
	g.dl.InsertAfter(target, newItem)
}
func (g *GenericFixedDL) InsertBefore(target *GenericNodeDL, newItem *GenericNodeDL) {
	g.dl.InsertBefore(target, newItem)
}
