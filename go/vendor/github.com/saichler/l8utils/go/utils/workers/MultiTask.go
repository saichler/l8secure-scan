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

package workers

import "sync"

type MultiTask struct {
	tasks   map[int]*Task
	results map[int]interface{}
	cond    *sync.Cond
}

type Task struct {
	task      ITask
	index     int
	multiTask *MultiTask
}

type ITask interface {
	Run() interface{}
}

func NewMultiTask() *MultiTask {
	mt := &MultiTask{}
	mt.tasks = make(map[int]*Task)
	mt.results = make(map[int]interface{})
	mt.cond = sync.NewCond(&sync.Mutex{})
	return mt
}

func (this *MultiTask) AddTask(t ITask) {
	this.cond.L.Lock()
	defer this.cond.L.Unlock()
	task := &Task{task: t, index: len(this.tasks), multiTask: this}
	this.tasks[len(this.tasks)] = task
}

func (this *MultiTask) RunAll() map[int]interface{} {
	this.cond.L.Lock()
	defer this.cond.L.Unlock()
	for _, task := range this.tasks {
		go task.Run()
	}
	this.cond.Wait()
	return this.results
}

func (this *Task) Run() {
	this.multiTask.cond.L.Lock()
	this.multiTask.cond.L.Unlock()

	err := this.task.Run()

	this.multiTask.cond.L.Lock()
	defer this.multiTask.cond.L.Unlock()
	this.multiTask.results[this.index] = err
	if len(this.multiTask.tasks) == len(this.multiTask.results) {
		this.multiTask.cond.Broadcast()
	}
}
