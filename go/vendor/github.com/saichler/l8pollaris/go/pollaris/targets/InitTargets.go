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

// Package targets provides target lifecycle management for the Layer 8 ecosystem.
// It handles the creation, persistence, and distribution of polling targets
// to collectors using PostgreSQL for storage and round-robin load balancing
// for target distribution.
package targets

import (
	"bytes"
	"fmt"
	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8ql/go/gsql/interpreter"
	"github.com/saichler/l8types/go/ifs"
	"strconv"
	"time"
)

// InitTargets initializes all targets from the database on service startup.
// It runs as a goroutine after a 30-second delay to allow the system to stabilize.
// Only the service leader performs initialization to avoid duplicate operations.
// The method:
// 1. Queries all targets from the database in pages of 500
// 2. Validates IP addresses for each target
// 3. First multicasts a DOWN state to all collectors (to clear stale state)
// 4. Then uses round-robin to distribute UP targets across available collectors
func (this *TargetCallback) InitTargets(vnic ifs.IVNic) {
	time.Sleep(time.Second * 30)
	leader := vnic.Resources().Services().GetLeader(ServiceName, ServiceArea)
	if leader != vnic.Resources().SysConfig().LocalUuid {
		fmt.Println("Not the leader of this service:", leader)
		return
	}
	gsql := "select * from L8PTarget limit 500 page "
	page := 0
	upTargets := make([]*l8tpollaris.L8PTarget, 0)
	for {
		buff := bytes.Buffer{}
		buff.WriteString(gsql)
		buff.WriteString(strconv.Itoa(page))
		q, e := interpreter.NewQuery(buff.String(), vnic.Resources())
		if e != nil {
			panic(e)
		}
		resp := this.iorm.Read(q, vnic.Resources())
		if resp.Error() != nil {
			break
		}
		if resp.Elements() == nil || len(resp.Elements()) == 0 {
			break
		}
		if resp.Element() == nil {
			break
		}
		for _, elem := range resp.Elements() {
			item := elem.(*l8tpollaris.L8PTarget)
			this.validateNewIP(item)
			if item.State == l8tpollaris.L8PTargetState_Up {
				upTargets = append(upTargets, item)
			}
		}
		page++
	}

	cService := ""
	cArea := byte(0)
	for _, item := range upTargets {
		if cService == "" {
			cService, cArea = Links.Collector(item.LinksId)
		}
		item.State = l8tpollaris.L8PTargetState_Down
		vnic.Multicast(cService, cArea, ifs.POST, item)
	}
	fmt.Println("Round Robin for ", len(upTargets), " targets")
	roundRobin := health.NewRoundRobin(cService, cArea, vnic.Resources())
	for _, item := range upTargets {
		item.State = l8tpollaris.L8PTargetState_Up
		next := roundRobin.Next()
		vnic.Unicast(next, cService, cArea, ifs.POST, item)
	}
}
