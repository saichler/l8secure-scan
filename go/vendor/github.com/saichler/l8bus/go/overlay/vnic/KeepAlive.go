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
	"time"

	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8types/go/ifs"
)

// KeepAlive monitors connection health and periodically sends health status updates.
// It tracks CPU and memory usage and broadcasts health statistics to the network.
type KeepAlive struct {
	vnic       *VirtualNetworkInterface
	startTime  int64
	cpuTracker *health.CPUTracker
}

func newKeepAlive(vnic *VirtualNetworkInterface) *KeepAlive {
	return &KeepAlive{
		vnic:       vnic,
		cpuTracker: &health.CPUTracker{},
	}
}

func (this *KeepAlive) start() {
	go this.run()
}
func (this *KeepAlive) shutdown() {}
func (this *KeepAlive) name() string {
	return "KA"
}
func (this *KeepAlive) run() {
	this.startTime = time.Now().UnixMilli()
	if this.vnic.resources.SysConfig().KeepAliveIntervalSeconds == 0 {
		return
	}
	// Send first keepalive after 1 second for fast initial health propagation
	time.Sleep(time.Second)
	if !this.vnic.running {
		return
	}
	this.sendState()
	for this.vnic.running {
		for i := 0; i < int(this.vnic.resources.SysConfig().KeepAliveIntervalSeconds*10); i++ {
			time.Sleep(time.Millisecond * 100)
			if !this.vnic.running {
				return
			}
		}
		this.sendState()
	}
}

func (this *KeepAlive) sendState() {
	hp := health.BaseHealthStats(this.vnic.resources)

	hp.Stats.TxMsgCount = this.vnic.healthStatistics.TxMsgCount.Load()
	hp.Stats.TxDataCount = this.vnic.healthStatistics.TxDataCount.Load()
	hp.Stats.RxMsgCount = this.vnic.healthStatistics.RxMsgCount.Load()
	hp.Stats.RxDataCont = this.vnic.healthStatistics.RxDataCont.Load()
	hp.Stats.LastMsgTime = this.vnic.healthStatistics.LastMsgTime.Load()

	hp.StartTime = this.startTime

	//this.vnic.resources.Logger().Debug("Sending Keep Alive for ", this.vnic.resources.SysConfig().LocalUuid, " ", this.vnic.resources.SysConfig().LocalAlias)
	//We unicast to the vnet, it will multicast the change to all
	this.vnic.Unicast(this.vnic.resources.SysConfig().RemoteUuid,
		health.ServiceName, health.ServiceArea(this.vnic.resources), ifs.PATCH, hp)
}
