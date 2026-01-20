package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCache_SetAndGet(t *testing.T) {
	c := New(time.Hour)

	c.Set("key1", "value1")

	val, found := c.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "value1", val)
}

func TestCache_GetMissing(t *testing.T) {
	c := New(time.Hour)

	val, found := c.Get("nonexistent")
	assert.False(t, found)
	assert.Nil(t, val)
}

func TestCache_Expiration(t *testing.T) {
	c := New(50 * time.Millisecond)

	c.Set("key", "value")

	// Should exist immediately
	val, found := c.Get("key")
	assert.True(t, found)
	assert.Equal(t, "value", val)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	val, found = c.Get("key")
	assert.False(t, found)
	assert.Nil(t, val)
}

func TestCache_SetWithTTL(t *testing.T) {
	c := New(time.Hour)

	c.SetWithTTL("short", "value", 50*time.Millisecond)
	c.Set("long", "value") // Uses default TTL of 1 hour

	// Both should exist
	_, found := c.Get("short")
	assert.True(t, found)
	_, found = c.Get("long")
	assert.True(t, found)

	// Wait for short TTL
	time.Sleep(100 * time.Millisecond)

	// Short should be expired, long should still exist
	_, found = c.Get("short")
	assert.False(t, found)
	_, found = c.Get("long")
	assert.True(t, found)
}

func TestCache_Delete(t *testing.T) {
	c := New(time.Hour)

	c.Set("key", "value")
	c.Delete("key")

	_, found := c.Get("key")
	assert.False(t, found)
}

func TestCache_Clear(t *testing.T) {
	c := New(time.Hour)

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Clear()

	_, found := c.Get("key1")
	assert.False(t, found)
	_, found = c.Get("key2")
	assert.False(t, found)
}

func TestCache_GetOrSet(t *testing.T) {
	c := New(time.Hour)

	callCount := 0
	fn := func() (interface{}, error) {
		callCount++
		return "computed", nil
	}

	// First call should invoke the function
	val, err := c.GetOrSet("key", fn)
	assert.NoError(t, err)
	assert.Equal(t, "computed", val)
	assert.Equal(t, 1, callCount)

	// Second call should use cached value
	val, err = c.GetOrSet("key", fn)
	assert.NoError(t, err)
	assert.Equal(t, "computed", val)
	assert.Equal(t, 1, callCount) // Function not called again
}

func TestMetricsCache(t *testing.T) {
	mc := NewMetricsCache()

	mc.Set(KeyCPU, "cpu-data")
	mc.Set(KeyMemory, "memory-data")

	val, found := mc.Get(KeyCPU)
	assert.True(t, found)
	assert.Equal(t, "cpu-data", val)

	val, found = mc.Get(KeyMemory)
	assert.True(t, found)
	assert.Equal(t, "memory-data", val)
}

func TestCache_ConcurrentAccess(t *testing.T) {
	c := New(time.Hour)

	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			c.Set("key", i)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			c.Get("key")
		}
		done <- true
	}()

	// Wait for both
	<-done
	<-done
}
