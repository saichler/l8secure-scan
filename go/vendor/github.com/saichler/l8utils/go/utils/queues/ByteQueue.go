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

package queues

import (
	"math/bits"
	"sync"

	"github.com/saichler/l8types/go/ifs"
)

// ByteQueue is a priority-based byte slice queue with 8 priority levels (0-7).
// Higher priority items are dequeued first using O(1) bit operations.
// It supports backpressure when full and graceful shutdown.
type ByteQueue struct {
	name         string
	queues       [][][]byte
	priorityMask uint8
	rwMtx        *sync.RWMutex
	cond         *sync.Cond
	maxSize      int
	active       bool
	size         int
}

// NewByteQueue creates a new priority-based byte queue with the specified maximum size.
// The queue supports 8 priority levels (0-7) determined by the priority byte in each message.
func NewByteQueue(name string, maxSize int) *ByteQueue {
	bq := &ByteQueue{
		name:         name,
		maxSize:      maxSize,
		active:       true,
		priorityMask: 0,
		size:         0,
	}
	bq.rwMtx = &sync.RWMutex{}
	bq.cond = sync.NewCond(bq.rwMtx)
	bq.queues = make([][][]byte, ifs.P1+1)
	for i := range bq.queues {
		bq.queues[i] = make([][]byte, 0, 16)
	}
	return bq
}

// Add enqueues a byte slice at its designated priority level.
// Priority is extracted from the byte at ifs.PPriority position (upper 4 bits).
// Blocks if the queue is at maximum capacity until space is available.
func (this *ByteQueue) Add(data []byte) {
	this.rwMtx.Lock()
	defer this.rwMtx.Unlock()

	// Wait if queue is full using proper condition variable
	for this.size >= this.maxSize && this.active {
		this.cond.Wait()
	}

	if !this.active {
		return
	}

	priority := data[ifs.PPriority] >> 4
	if priority > 7 {
		priority = 7 // Cap at maximum priority
	}

	this.queues[priority] = append(this.queues[priority], data)
	this.priorityMask |= (1 << priority) // Set bit for this priority
	this.size++

	this.cond.Broadcast()
}

// Next dequeues and returns the highest priority item. Blocks if the queue is empty.
// Returns nil if the queue has been shut down.
func (this *ByteQueue) Next() []byte {
	for this.active {
		var item []byte
		this.rwMtx.Lock()
		if this.size == 0 {
			this.cond.Wait()
		} else {
			item = this.next()
		}
		this.rwMtx.Unlock()
		if item != nil {
			return item
		}
	}
	return nil
}

// Active returns true if the queue has not been shut down.
func (this *ByteQueue) Active() bool {
	return this.active
}

// Shutdown stops the queue, clears all pending items, and wakes blocked goroutines.
func (this *ByteQueue) Shutdown() {
	this.rwMtx.Lock()
	defer this.rwMtx.Unlock()

	this.active = false
	this.clear()
	this.cond.Broadcast()
}

func (this *ByteQueue) next() []byte {
	if this.priorityMask == 0 {
		return nil // No items in any queue
	}

	// Find highest priority with items - O(1) bit operation
	priority := 7 - bits.LeadingZeros8(this.priorityMask)

	// Dequeue from highest priority - O(1)
	queue := &this.queues[priority]
	item := (*queue)[0]
	(*queue)[0] = nil
	*queue = (*queue)[1:]
	this.size--

	// Clear bit if this priority queue becomes empty - O(1)
	if len(*queue) == 0 {
		this.priorityMask &^= (1 << priority)
	}

	this.cond.Broadcast() // Signal waiting producers
	return item
}

func (this *ByteQueue) clear() {
	for i := range this.queues {
		this.queues[i] = make([][]byte, 0) // Fresh slice
	}
	this.priorityMask = 0
	this.size = 0
}

// Size returns the total number of items across all priority levels.
func (this *ByteQueue) Size() int {
	this.rwMtx.RLock()
	defer this.rwMtx.RUnlock()
	return this.size
}

// Clear removes all items from all priority levels.
func (this *ByteQueue) Clear() {
	this.rwMtx.Lock()
	defer this.rwMtx.Unlock()
	this.clear()
}

// IsEmpty returns true if there are no items in any priority level.
func (this *ByteQueue) IsEmpty() bool {
	this.rwMtx.RLock()
	defer this.rwMtx.RUnlock()
	return this.size == 0
}
