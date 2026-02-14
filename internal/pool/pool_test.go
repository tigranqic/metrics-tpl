package pool

import (
	"testing"
)

// TestObject is a simple test object that implements Resetter interface.
type TestObject struct {
	value int
	name  string
	data  []int
}

// Reset implements the Resetter interface for TestObject.
func (to *TestObject) Reset() {
	to.value = 0
	to.name = ""
	to.data = to.data[:0]
}

func TestPoolGetPut(t *testing.T) {
	pool := New(func() *TestObject {
		return &TestObject{}
	})

	// Get an object from empty pool
	obj1 := pool.Get()
	if obj1 == nil {
		t.Fatal("Get() returned nil")
	}

	// Modify the object
	obj1.value = 42
	obj1.name = "test"
	obj1.data = []int{1, 2, 3}

	// Put the object back (should reset)
	pool.Put(obj1)

	// Get another object (should be the reset one)
	obj2 := pool.Get()
	if obj2 == nil {
		t.Fatal("Get() returned nil after Put")
	}

	// Verify fields are reset
	if obj2.value != 0 {
		t.Errorf("expected value=0, got %d", obj2.value)
	}
	if obj2.name != "" {
		t.Errorf("expected name=\"\", got %q", obj2.name)
	}
	if len(obj2.data) != 0 {
		t.Errorf("expected data length=0, got %d", len(obj2.data))
	}
}

func TestPoolMultipleObjects(t *testing.T) {
	pool := New(func() *TestObject {
		return &TestObject{}
	})

	// Get multiple objects
	obj1 := pool.Get()
	obj1.value = 10

	obj2 := pool.Get()
	obj2.value = 20

	// Verify they are different
	if obj1 == obj2 {
		t.Fatal("different Get() calls returned the same object")
	}

	// Put them back
	pool.Put(obj1)
	pool.Put(obj2)

	// Get them back
	obj3 := pool.Get()
	obj4 := pool.Get()

	// They should be reset
	if obj3.value != 0 {
		t.Errorf("obj3: expected value=0, got %d", obj3.value)
	}
	if obj4.value != 0 {
		t.Errorf("obj4: expected value=0, got %d", obj4.value)
	}
}

func TestPoolFactory(t *testing.T) {
	factory := func() *TestObject {
		return &TestObject{
			data: make([]int, 0, 10),
		}
	}

	pool := New(factory)

	obj1 := pool.Get()
	if obj1 == nil {
		t.Fatal("Get() returned nil")
	}
	if cap(obj1.data) != 10 {
		t.Errorf("expected cap=10, got %d", cap(obj1.data))
	}

	obj1.data = append(obj1.data, 1, 2, 3)
	pool.Put(obj1)

	obj2 := pool.Get()
	if obj2 == nil {
		t.Fatal("Get() returned nil")
	}
	if len(obj2.data) != 0 {
		t.Errorf("expected len=0, got %d", len(obj2.data))
	}
	if cap(obj2.data) != 10 {
		t.Errorf("expected cap=10, got %d", cap(obj2.data))
	}
}

func BenchmarkPoolGetPut(b *testing.B) {
	pool := New(func() *TestObject {
		return &TestObject{
			data: make([]int, 0, 100),
		}
	})

	b.ResetTimer()
	for b.Loop() {
		obj := pool.Get()
		obj.value = 42
		obj.name = "test"
		obj.data = append(obj.data, 1, 2, 3)
		pool.Put(obj)
	}
}
