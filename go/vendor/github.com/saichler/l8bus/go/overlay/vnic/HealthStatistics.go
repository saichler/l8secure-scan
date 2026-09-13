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

package vnic

import (
	"sync/atomic"
	"time"
)

// HealthStatistics tracks message and data transfer statistics for health monitoring.
// All counters are thread-safe using atomic operations.
type HealthStatistics struct {
	LastMsgTime atomic.Int64
	TxMsgCount  atomic.Int64
	TxDataCount atomic.Int64
	RxMsgCount  atomic.Int64
	RxDataCont  atomic.Int64
}

// Stamp updates the last message timestamp to the current time.
func (this *HealthStatistics) Stamp() {
	this.LastMsgTime.Store(time.Now().UnixMilli())
}

// IncrementTX increments the transmitted message count and data byte count.
func (this *HealthStatistics) IncrementTX(data []byte) {
	this.TxMsgCount.Add(1)
	this.TxDataCount.Add(int64(len(data)))
}

// IncrementRx increments the received message count and data byte count.
func (this *HealthStatistics) IncrementRx(data []byte) {
	this.RxMsgCount.Add(1)
	this.RxDataCont.Add(int64(len(data)))
}
