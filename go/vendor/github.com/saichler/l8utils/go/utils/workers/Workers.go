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

// Package workers provides a simple worker pool for concurrent task execution with
// configurable concurrency limits. It ensures that no more than a specified number
// of workers run simultaneously.
//
// Key features:
//   - Configurable maximum concurrent workers
//   - Automatic worker coordination using condition variables
//   - Simple IWorker interface for task implementation
package workers

import "sync"

// Workers manages a pool of concurrent workers with a maximum limit.
// It uses a condition variable to block new workers when at capacity.
type Workers struct {
	limit   int
	running int
	cond    *sync.Cond
}

// IWorker defines the interface for tasks that can be executed by the worker pool.
type IWorker interface {
	Run()
}

// Worker wraps an IWorker with a reference to its parent pool for cleanup.
type Worker struct {
	worker  IWorker
	workers *Workers
}

// NewWorkers creates a new worker pool with the specified maximum concurrency limit.
func NewWorkers(limit int) *Workers {
	return &Workers{limit: limit, cond: sync.NewCond(&sync.Mutex{})}
}

func (this *Workers) canStart() {
	this.cond.L.Lock()
	defer this.cond.L.Unlock()
	for this.running >= this.limit {
		this.cond.Wait()
	}
	this.running++
}

// Run submits a worker for execution. Blocks if the pool is at capacity until a slot opens.
func (this *Workers) Run(worker IWorker) {
	this.canStart()
	w := &Worker{worker: worker, workers: this}
	go w.run()
}

func (this Worker) run() {
	this.worker.Run()
	this.workers.cond.L.Lock()
	defer this.workers.cond.L.Unlock()
	this.workers.running--
	this.workers.cond.Broadcast()
}
