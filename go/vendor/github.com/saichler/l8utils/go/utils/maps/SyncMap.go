// © 2025 Sharon Aicler (saichler@gmail.com)
//
// Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package maps provides thread-safe map implementations for concurrent access.
// SyncMap wraps a standard Go map with sync.RWMutex for safe concurrent reads and writes.
//
// Key features:
//   - Thread-safe Put, Get, Delete, and Contains operations
//   - Size tracking and iteration support
//   - Nil-safe operations (methods handle nil receiver gracefully)
//   - ValuesAsList and KeysAsList for extracting typed slices with optional filtering
package maps

import (
	"reflect"
	"sync"
)

// SyncMap is a thread-safe map implementation using sync.RWMutex.
// All operations are safe for concurrent access from multiple goroutines.
type SyncMap struct {
	m map[interface{}]interface{}
	s *sync.RWMutex
}

// NewSyncMap creates a new empty thread-safe map.
func NewSyncMap() *SyncMap {
	mm := &SyncMap{}
	mm.m = make(map[interface{}]interface{})
	mm.s = &sync.RWMutex{}
	return mm
}

// Put stores a key-value pair. Returns true if this is a new key, false if updating existing.
func (this *SyncMap) Put(key, value interface{}) bool {
	if this == nil {
		return false
	}
	this.s.Lock()
	defer this.s.Unlock()
	_, ok := this.m[key]
	this.m[key] = value
	return !ok
}

// PutIfAbsent inserts the key/value pair only when the key is not already present.
// Returns true if the value was inserted, false if the key already had a value
// (which is preserved untouched). The check-and-insert is atomic under the
// map's lock.
func (this *SyncMap) PutIfAbsent(key, value interface{}) bool {
	if this == nil {
		return false
	}
	this.s.Lock()
	defer this.s.Unlock()
	if _, ok := this.m[key]; ok {
		return false
	}
	this.m[key] = value
	return true
}

// Get retrieves a value by key. Returns the value and whether it was found.
func (this *SyncMap) Get(key interface{}) (interface{}, bool) {
	if this == nil {
		return nil, false
	}
	this.s.RLock()
	defer this.s.RUnlock()
	v, ok := this.m[key]
	return v, ok
}

// Contains returns true if the key exists in the map.
func (this *SyncMap) Contains(key interface{}) bool {
	if this == nil {
		return false
	}
	this.s.RLock()
	defer this.s.RUnlock()
	_, ok := this.m[key]
	return ok
}

// Delete removes a key and returns its value and whether it existed.
func (this *SyncMap) Delete(key interface{}) (interface{}, bool) {
	if this == nil {
		return nil, false
	}
	this.s.Lock()
	defer this.s.Unlock()
	v, ok := this.m[key]
	delete(this.m, key)
	return v, ok
}

// Size returns the number of entries in the map.
func (this *SyncMap) Size() int {
	if this == nil {
		return 0
	}
	this.s.RLock()
	defer this.s.RUnlock()
	return len(this.m)
}

// Clean removes all entries and returns the old map contents.
func (this *SyncMap) Clean() map[interface{}]interface{} {
	if this == nil {
		return nil
	}
	this.s.Lock()
	defer this.s.Unlock()
	result := this.m
	this.m = make(map[interface{}]interface{})
	return result
}

// ValuesAsList returns map values as a typed slice. Optional filter excludes non-matching values.
func (this *SyncMap) ValuesAsList(typ reflect.Type, filter func(interface{}) bool) interface{} {
	if this == nil {
		return false
	}
	this.s.RLock()
	defer this.s.RUnlock()
	newSlice := reflect.MakeSlice(reflect.SliceOf(typ), len(this.m), len(this.m))
	index := 0
	for _, v := range this.m {
		if filter != nil && !filter(v) {
			continue
		}
		newSlice.Index(index).Set(reflect.ValueOf(v))
		index++
	}

	if index < len(this.m) {
		filterSlice := reflect.MakeSlice(reflect.SliceOf(typ), index, index)
		for i := 0; i < index; i++ {
			filterSlice.Index(i).Set(newSlice.Index(i))
		}
		return filterSlice.Interface()
	}

	return newSlice.Interface()
}

// KeysAsList returns map keys as a typed slice. Optional filter excludes non-matching keys.
func (this *SyncMap) KeysAsList(typ reflect.Type, filter func(interface{}) bool) interface{} {
	if this == nil {
		return false
	}
	this.s.RLock()
	defer this.s.RUnlock()
	newSlice := reflect.MakeSlice(reflect.SliceOf(typ), len(this.m), len(this.m))
	index := 0
	for v, _ := range this.m {
		if filter != nil && !filter(v) {
			continue
		}
		newSlice.Index(index).Set(reflect.ValueOf(v))
		index++
	}

	if index < len(this.m) {
		filterSlice := reflect.MakeSlice(reflect.SliceOf(typ), index, index)
		for i := 0; i < index; i++ {
			filterSlice.Index(i).Set(newSlice.Index(i))
		}
		return filterSlice.Interface()
	}

	return newSlice.Interface()
}

// Iterate calls the provided function for each key-value pair while holding a read lock.
func (this *SyncMap) Iterate(do func(k, v interface{})) {
	if this == nil {
		return
	}
	this.s.RLock()
	defer this.s.RUnlock()
	for k, v := range this.m {
		do(k, v)
	}
}
