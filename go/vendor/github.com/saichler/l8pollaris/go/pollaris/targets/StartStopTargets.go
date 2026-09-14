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

package targets

import (
	"bytes"
	"fmt"
	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8ql/go/gsql/interpreter"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/strings"
	"strconv"
	"time"
)

// startStopAll performs bulk start or stop operations on all targets of a specific type.
// Only the service leader executes this operation to ensure consistency.
// The method:
// 1. Queries targets by inventory type and current state (opposite of desired state)
// 2. Updates target states in the database in batches of 500
// 3. For UP state: uses round-robin to distribute targets across collectors
// 4. For DOWN state: multicasts to all collectors to stop the target
// A small delay (10 microseconds) is added between sends to prevent overwhelming collectors.
func (this *TargetCallback) startStopAll(state l8tpollaris.L8PTargetState, typ l8tpollaris.L8PTargetType, vnic ifs.IVNic) {
	leader := vnic.Resources().Services().GetLeader(ServiceName, ServiceArea)
	if leader != vnic.Resources().SysConfig().LocalUuid {
		return
	}

	gsql := strings.New("select * from L8PTarget where InventoryType=")
	gsql.Add(gsql.StringOf(typ))
	gsql.Add(" and (State=0 or State=")
	switch state {
	case l8tpollaris.L8PTargetState_Up:
		gsql.Add(strconv.Itoa(int(l8tpollaris.L8PTargetState_Down)))
	case l8tpollaris.L8PTargetState_Down:
		gsql.Add(strconv.Itoa(int(l8tpollaris.L8PTargetState_Up)))
	default:
		vnic.Resources().Logger().Error("Not Supported Target State ", state.String())
		return
	}
	gsql.Add(") limit 500 page ")
	fmt.Println("query is:", gsql.String())

	collectorService := ""
	collectorArea := byte(0)
	page := 0
	targets := make([]*l8tpollaris.L8PTarget, 0)
	for {
		buff := bytes.Buffer{}
		buff.Write(gsql.Bytes())
		buff.WriteString(strconv.Itoa(page))
		q, e := interpreter.NewQuery(buff.String(), vnic.Resources())
		if e != nil {
			panic(e)
		}
		resp := this.iorm.Read(q, vnic.Resources())
		if resp.Error() != nil {
			fmt.Println(resp.Error().Error())
			break
		}
		if resp.Elements() == nil || len(resp.Elements()) == 0 {
			fmt.Println("Empty Result")
			break
		}
		if resp.Element() == nil {
			fmt.Println("Element is nil")
			break
		}
		fmt.Println("Size of elements=", len(resp.Elements()))
		for _, elem := range resp.Elements() {
			item := elem.(*l8tpollaris.L8PTarget)
			item.State = state
			targets = append(targets, item)
			if collectorService == "" {
				collectorService, collectorArea = Links.Collector(item.LinksId)
			}
		}
		page++
	}

	fmt.Println("Sending start/stop to ", collectorService, " ", collectorArea)

	bulk := make([]*l8tpollaris.L8PTarget, 0)
	for _, target := range targets {
		bulk = append(bulk, target)
		if len(bulk) >= 500 {
			elems := object.New(nil, bulk)
			err := this.iorm.Write(ifs.PATCH, elems, vnic.Resources())
			if err != nil {
				vnic.Resources().Logger().Error(err.Error())
			}
			bulk = make([]*l8tpollaris.L8PTarget, 0)
		}
	}

	if len(bulk) > 0 {
		elems := object.New(nil, bulk)
		err := this.iorm.Write(ifs.PATCH, elems, vnic.Resources())
		if err != nil {
			vnic.Resources().Logger().Error(err.Error())
		}
	}

	roundRobin := health.NewRoundRobin(collectorService, collectorArea, vnic.Resources())
	for _, target := range targets {
		time.Sleep(time.Microsecond)
		switch target.State {
		case l8tpollaris.L8PTargetState_Up:
			next := roundRobin.Next()
			vnic.Unicast(next, collectorService, collectorArea, ifs.POST, target)
		case l8tpollaris.L8PTargetState_Down:
			vnic.Multicast(collectorService, collectorArea, ifs.POST, target)
		}
	}
}
