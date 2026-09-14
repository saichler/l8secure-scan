/*
 * © 2025 Sharon Aicler (saichler@gmail.com)
 *
 * Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package t_topology

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8bus/go/overlay/protocol"
	. "github.com/saichler/l8bus/go/overlay/vnet"
	. "github.com/saichler/l8test/go/infra/t_resources"
	. "github.com/saichler/l8test/go/infra/t_service"
	. "github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8health"
)

type TestTopology struct {
	vnets       map[string]*VNet
	vnetsOrder  []*VNet
	vnics       map[string]IVNic
	handlers    map[string]*TestServiceHandler
	trHandlers  map[string]*TestServiceTransactionHandler
	repHandlers map[string]*TestServiceReplicationHandler
	mtx         *sync.RWMutex
}

func NewTestTopology(vnicCountPervNet int, vnetPorts []int, level LogLevel) *TestTopology {
	protocol.Discovery_Enabled = false
	this := &TestTopology{}
	this.vnets = make(map[string]*VNet)
	this.vnics = make(map[string]IVNic)
	this.handlers = make(map[string]*TestServiceHandler)
	this.trHandlers = make(map[string]*TestServiceTransactionHandler)
	this.repHandlers = make(map[string]*TestServiceReplicationHandler)
	this.vnetsOrder = make([]*VNet, 0)
	this.mtx = &sync.RWMutex{}

	Log.Info("Creating Vnets...")
	for _, vNetPort := range vnetPorts {
		_vnet := createVnet(vNetPort, level)
		this.vnets[_vnet.Resources().SysConfig().LocalAlias] = _vnet
		this.vnetsOrder = append(this.vnetsOrder, _vnet)
	}

	Log.Info("Connecting Vnets...")
	for i := 0; i < len(this.vnetsOrder)-1; i++ {
		for j := i + 1; j < len(this.vnetsOrder); j++ {
			connectVnets(this.vnetsOrder[i], this.vnetsOrder[j])
		}
	}

	Log.Info("Waiting for Vnets topology...")
	if !WaitForCondition(this.areVnetsConnected, 2, nil, "Vnets are not ready and connected") {
		panic("Vnets are not ready and connects ")
	}

	Log.Info("Creating Vnics...")
	for _, vNetPort := range vnetPorts {
		for i := 0; i < vnicCountPervNet; i++ {
			if i == vnicCountPervNet-1 {
				_vnic, _, _, _ := createVnic(vNetPort, i+1, -1, level)
				this.vnics[_vnic.Resources().SysConfig().LocalAlias] = _vnic
			} else {
				_vnic, handler, trHandler, repHandler := createVnic(vNetPort, i+1, 0, level)
				this.vnics[_vnic.Resources().SysConfig().LocalAlias] = _vnic
				this.handlers[_vnic.Resources().SysConfig().LocalAlias] = handler
				this.trHandlers[_vnic.Resources().SysConfig().LocalAlias] = trHandler
				this.repHandlers[_vnic.Resources().SysConfig().LocalAlias] = repHandler
			}
		}
	}

	Log.Info("Waiting for vnets health...")
	if !WaitForCondition(this.areVnetsHaveAllVnics, 2, nil, "Vnet are not ready 2") {
		for _, vnet := range this.vnets {
			hc, _ := health.HealthServiceCache(vnet.Resources())
			fmt.Println(vnet.Resources().SysConfig().LocalAlias, " ", vnet.ExternalCount(), vnet.LocalCount(), hc.Size())
		}
		panic("Vnets health is not ready")
	}

	Log.Info("Waiting for vnics health...")
	if !WaitForCondition(this.areVnicsReady, 2, nil, "Vnics are not ready!") {
		vnicName := ""
		vnicSum := 0
		hc := make(map[string]*l8health.L8Health)
		for vnetNum := 1; vnetNum <= 3; vnetNum++ {
			for vnicNum := 1; vnicNum <= 4; vnicNum++ {
				nic := this.VnicByVnetNum(vnetNum, vnicNum)
				hc = health.All(nic.Resources())
				if len(hc) < 15 {
					vnicName = nic.Resources().SysConfig().LocalAlias
					vnicSum = len(hc)
					break
				}
			}
			if vnicName != "" {
				break
			}
		}
		for _, hp := range hc {
			fmt.Println(" - ", hp.Alias, " - ")
		}
		panic("Vnics are not ready, vnic " + vnicName + " has only " + strconv.Itoa(vnicSum) + " instead of 15")
	}

	Log.Info("Waiting for test services in health...")
	if !WaitForCondition(this.areVnicsServicesReady, 5, nil, "Vnics are not ready!") {
		panic("Vnic Test Services are not ready")
	}

	Log.Info("Waiting for test services leader & participants...")
	if !WaitForCondition(this.areVnicsServicesTransactionReady, 20, nil, "Vnics are not ready!") {
		for vnetNum := 1; vnetNum <= 3; vnetNum++ {
			for vnicNum := 1; vnicNum <= 3; vnicNum++ {
				nic := this.VnicByVnetNum(vnetNum, vnicNum)
				if nic.Resources().Services().GetLeader("Tests", 1) == "" {
					fmt.Println("vNic ", nic.Resources().SysConfig().LocalAlias, " Does not have a leader for Tests")
				}
				participants := len(nic.Resources().Services().GetParticipants("Tests", 1))
				if participants != 9 {
					fmt.Println("vNic ", nic.Resources().SysConfig().LocalAlias, " Participants ", participants)
				}
			}
		}
		panic("Vnic Test Services Transactions are not ready")
	}

	Log.Info("Waiting for MapReduce participants...")
	if !WaitForCondition(this.areMapReduceParticipantsReady, 20, nil, "MapReduce participants are not ready!") {
		for vnetNum := 1; vnetNum <= 3; vnetNum++ {
			for vnicNum := 1; vnicNum <= 3; vnicNum++ {
				nic := this.VnicByVnetNum(vnetNum, vnicNum)
				participants := len(nic.Resources().Services().GetParticipants(ServiceName, 0))
				if participants != 9 {
					fmt.Println("vNic ", nic.Resources().SysConfig().LocalAlias, " MapReduce Participants ", participants)
				}
			}
		}
		panic("MapReduce participants are not ready")
	}

	return this
}

func (this *TestTopology) areVnicsReady() bool {
	for vnetNum := 1; vnetNum <= 3; vnetNum++ {
		for vnicNum := 1; vnicNum <= 4; vnicNum++ {
			nic := this.VnicByVnetNum(vnetNum, vnicNum)
			hc, _ := health.HealthServiceCache(nic.Resources())
			if hc.Size() < 15 {
				return false
			}
		}
	}
	return true
}

func (this *TestTopology) areVnicsServicesReady() bool {
	for vnetNum := 1; vnetNum <= 3; vnetNum++ {
		for vnicNum := 1; vnicNum <= 4; vnicNum++ {
			nic := this.VnicByVnetNum(vnetNum, vnicNum)
			hps := health.All(nic.Resources())
			count := 0
			for _, hp := range hps {
				for srv, _ := range hp.Services.ServiceToAreas {
					if srv == ServiceName {
						count++
					}
				}
			}
			if count < 9 {
				return false
			}
		}
	}

	return true
}

func (this *TestTopology) areVnicsServicesTransactionReady() bool {
	for vnetNum := 1; vnetNum <= 3; vnetNum++ {
		for vnicNum := 1; vnicNum <= 3; vnicNum++ {
			nic := this.VnicByVnetNum(vnetNum, vnicNum)
			if nic.Resources().Services().GetLeader("Tests", 1) == "" {
				return false
			}
			participants := len(nic.Resources().Services().GetParticipants("Tests", 1))
			if participants != 9 {
				return false
			}
		}
	}

	return true
}

func (this *TestTopology) areMapReduceParticipantsReady() bool {
	for vnetNum := 1; vnetNum <= 3; vnetNum++ {
		for vnicNum := 1; vnicNum <= 3; vnicNum++ {
			nic := this.VnicByVnetNum(vnetNum, vnicNum)
			participants := len(nic.Resources().Services().GetParticipants(ServiceName, 0))
			if participants != 9 {
				return false
			}
		}
	}
	return true
}

func (this *TestTopology) areVnetsConnected() bool {
	for _, vnet := range this.vnets {
		if vnet.ExternalCount() != 2 {
			return false
		}
	}
	return true
}

func (this *TestTopology) areVnetsHaveAllVnics() bool {
	for _, vnet := range this.vnets {
		if vnet.ExternalCount() != 2 {
			return false
		}
		if vnet.LocalCount() != 4 {
			return false
		}
		hc, ok := health.HealthServiceCache(vnet.Resources())
		if hc == nil || !ok {
			hc, ok = health.HealthServiceCache(vnet.Resources())
			panic("nil health cache")
		}
		if hc.Size() < 15 {
			return false
		}
	}
	return true
}

func (this *TestTopology) Vnet(vnetPort int) *VNet {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	alias := AliasOf(vnetPort, -1)
	return this.vnets[alias]
}

func (this *TestTopology) VnicByPort(vnetPort, vnicNum int) IVNic {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	alias := AliasOf(vnetPort, vnicNum)
	return this.vnics[alias]
}

func (this *TestTopology) VnicByVnetNum(vnetNum, vnicNum int) IVNic {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	vnetPort := int(this.vnetsOrder[vnetNum-1].Resources().SysConfig().VnetPort)
	alias := AliasOf(vnetPort, vnicNum)
	return this.vnics[alias]
}

func (this *TestTopology) HandlerByPort(vnetPort, vnicNum int) *TestServiceHandler {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	alias := AliasOf(vnetPort, vnicNum)
	return this.handlers[alias]
}

func (this *TestTopology) HandlerByVnetNum(vnetNum, vnicNum int) *TestServiceHandler {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	vnetPort := int(this.vnetsOrder[vnetNum-1].Resources().SysConfig().VnetPort)
	alias := AliasOf(vnetPort, vnicNum)
	return this.handlers[alias]
}

func (this *TestTopology) TrHandlerByVnetNum(vnetNum, vnicNum int) *TestServiceTransactionHandler {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	vnetPort := int(this.vnetsOrder[vnetNum-1].Resources().SysConfig().VnetPort)
	alias := AliasOf(vnetPort, vnicNum)
	return this.trHandlers[alias]
}

func (this *TestTopology) RepHandlerByVnetNum(vnetNum, vnicNum int) *TestServiceReplicationHandler {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	vnetPort := int(this.vnetsOrder[vnetNum-1].Resources().SysConfig().VnetPort)
	alias := AliasOf(vnetPort, vnicNum)
	return this.repHandlers[alias]
}

func (this *TestTopology) Shutdown() {
	for _, _vnic := range this.vnics {
		_vnic.Shutdown()
	}
	for _, _vnet := range this.vnets {
		_vnet.Shutdown()
	}
}

func (this *TestTopology) ResetHandlers() {
	for _, _handler := range this.handlers {
		_handler.Reset()
	}
	for _, _handler := range this.trHandlers {
		_handler.Reset()
	}
	for _, _handler := range this.repHandlers {
		_handler.Reset()
	}
}

func (this *TestTopology) AllHandlers() []*TestServiceHandler {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	result := make([]*TestServiceHandler, 0)
	for _, h := range this.handlers {
		result = append(result, h)
	}
	return result
}

func (this *TestTopology) AllTrHandlers() []*TestServiceTransactionHandler {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	result := make([]*TestServiceTransactionHandler, 0)
	for _, h := range this.trHandlers {
		result = append(result, h)
	}
	return result
}

func (this *TestTopology) AllRepHandlers() []*TestServiceReplicationHandler {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	result := make([]*TestServiceReplicationHandler, 0)
	for _, h := range this.repHandlers {
		result = append(result, h)
	}
	return result
}

func (this *TestTopology) AllVnics() []IVNic {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	result := make([]IVNic, 0)
	for _, vnic := range this.vnics {
		result = append(result, vnic)
	}
	return result
}

func (this *TestTopology) RenewVnic(vnetNum, vnicNum int) {
	nic := this.VnicByVnetNum(vnetNum, vnicNum)
	oldSysConfig := nic.Resources().SysConfig()
	nic.Shutdown()
	_vnic, handler, trHandler, repHandler := createVnic(int(oldSysConfig.VnetPort), vnicNum, 0, Debug_Level)
	this.mtx.Lock()
	defer this.mtx.Unlock()

	this.vnics[_vnic.Resources().SysConfig().LocalAlias] = _vnic
	this.handlers[_vnic.Resources().SysConfig().LocalAlias] = handler
	this.trHandlers[_vnic.Resources().SysConfig().LocalAlias] = trHandler
	this.repHandlers[_vnic.Resources().SysConfig().LocalAlias] = repHandler
}

func (this *TestTopology) SetLogLevel(lvl LogLevel) {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	for _, net := range this.vnets {
		net.Resources().Logger().SetLogLevel(lvl)
	}
	for _, nic := range this.vnics {
		nic.Resources().Logger().SetLogLevel(lvl)
	}
}

func (this *TestTopology) ReActivateTestService(nic IVNic) {
	sla := NewServiceLevelAgreement(&TestServiceHandler{}, ServiceName, 0, false, nil)
	sla.SetArgs(nic.Resources().SysConfig().LocalAlias)
	h, err := nic.Resources().Services().Activate(sla, nic)
	if err != nil {
		panic(err)
	}
	handler := h.(*TestServiceHandler)
	this.handlers[nic.Resources().SysConfig().LocalAlias] = handler
}
