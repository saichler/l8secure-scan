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

package vnet

import (
	"sync"
	"sync/atomic"

	"github.com/saichler/l8types/go/ifs"
)

// Connections manages all VNic connections for a VNet, categorized as internal
// (same network), external VNet (cross-network switches), or external VNic (direct endpoints).
// It provides thread-safe access to connections and maintains size counters.
type Connections struct {
	internal         *sync.Map
	externalVnet     *sync.Map
	externalVnic     *sync.Map
	routeTable       *RouteTable
	logger           ifs.ILogger
	vnetUuid         string
	sizeInternal     atomic.Int32
	sizeExternalVnet atomic.Int32
	sizeExternalVnic atomic.Int32
}

// newConnections creates a new Connections manager for the given VNet UUID.
func newConnections(vnetUuid string, routeTable *RouteTable, logger ifs.ILogger) *Connections {
	conns := &Connections{}
	conns.internal = &sync.Map{}
	conns.externalVnet = &sync.Map{}
	conns.externalVnic = &sync.Map{}
	conns.routeTable = routeTable
	conns.logger = logger
	conns.vnetUuid = vnetUuid
	return conns
}

// addInternal registers an internal VNic connection, shutting down any existing connection with the same UUID.
func (this *Connections) addInternal(uuid string, vnic ifs.IVNic) {
	this.logger.Debug("Adding internal with alias ", vnic.Resources().SysConfig().RemoteAlias)
	exist, ok := this.internal.Load(uuid)
	if ok {
		this.logger.Debug("Internal Connection ", uuid, " already exists, shutting down")
		existVnic := exist.(ifs.IVNic)
		existVnic.Shutdown()
		this.internal.Delete(uuid)
	}
	this.internal.Store(uuid, vnic)
	this.sizeInternal.Add(1)
}

// addExternalVnet registers an external VNet connection, shutting down any existing connection with the same UUID.
func (this *Connections) addExternalVnet(uuid string, vnic ifs.IVNic) {
	this.logger.Debug("Adding external with alias ", vnic.Resources().SysConfig().RemoteAlias)
	exist, ok := this.externalVnet.Load(uuid)
	if ok {
		this.logger.Debug("External vnet ", uuid, " already exists, shutting down")
		existVnic := exist.(ifs.IVNic)
		existVnic.Shutdown()
		this.externalVnet.Delete(uuid)
	}
	this.externalVnet.Store(uuid, vnic)
	this.sizeExternalVnet.Add(1)
}

// addExternalVnic registers an external VNic connection, shutting down any existing connection with the same UUID.
func (this *Connections) addExternalVnic(uuid string, vnic ifs.IVNic) {
	this.logger.Debug("Adding external vnic with alias ", vnic.Resources().SysConfig().RemoteAlias)
	exist, ok := this.externalVnic.Load(uuid)
	if ok {
		this.logger.Debug("External vnic ", uuid, " already exists, shutting down")
		existVnic := exist.(ifs.IVNic)
		existVnic.Shutdown()
		this.externalVnic.Delete(uuid)
	}
	this.externalVnic.Store(uuid, vnic)
	this.sizeExternalVnic.Add(1)
}

// isConnected checks if there is an existing external VNet connection to the given IP address.
func (this *Connections) isConnected(ip string) bool {
	connected := false
	this.externalVnet.Range(func(key, value interface{}) bool {
		conn := value.(ifs.IVNic)
		addr := conn.Resources().SysConfig().Address
		if ip == addr {
			connected = true
			return false
		}
		return true
	})
	return connected
}

// getConnection retrieves a VNic by UUID, searching internal, external VNic, and route table.
// If isHope0 is true (message originates from this switch), it also checks external VNet connections.
func (this *Connections) getConnection(vnicUuid string, isHope0 bool) (string, ifs.IVNic) {
	//internal vnic
	vnic, ok := this.internal.Load(vnicUuid)
	if ok {
		return vnicUuid, vnic.(ifs.IVNic)
	}
	//external vnic
	vnic, ok = this.externalVnic.Load(vnicUuid)
	if ok {
		return vnicUuid, vnic.(ifs.IVNic)
	}
	// only if this is hope0, e.g. the source of the message is from this switch sources,
	// fetch try to find the route
	if isHope0 {
		vnic, ok = this.externalVnet.Load(vnicUuid)
		if ok {
			return vnicUuid, vnic.(ifs.IVNic)
		}
		remoteUuid := ""
		remoteUuid, ok = this.routeTable.vnetOf(vnicUuid)
		if !ok {
			return "", nil
		}
		vnic, ok = this.externalVnet.Load(remoteUuid)
		if ok {
			return remoteUuid, vnic.(ifs.IVNic)
		}
	}
	return "", nil
}

// all returns a map of all connections (internal and external VNet) keyed by UUID.
func (this *Connections) all() map[string]ifs.IVNic {
	all := make(map[string]ifs.IVNic)
	this.internal.Range(func(key, value interface{}) bool {
		all[key.(string)] = value.(ifs.IVNic)
		return true
	})
	this.externalVnet.Range(func(key, value interface{}) bool {
		all[key.(string)] = value.(ifs.IVNic)
		return true
	})
	return all
}

// isInterval checks if the given UUID corresponds to an internal connection.
func (this *Connections) isInterval(uuid string) bool {
	_, ok := this.internal.Load(uuid)
	return ok
}

// allInternals returns a map of all internal connections keyed by UUID.
func (this *Connections) allInternals() map[string]ifs.IVNic {
	result := make(map[string]ifs.IVNic)
	this.internal.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(ifs.IVNic)
		return true
	})
	return result
}

// allExternalVnets returns a map of all external VNet connections keyed by UUID.
func (this *Connections) allExternalVnets() map[string]ifs.IVNic {
	result := make(map[string]ifs.IVNic)
	this.externalVnet.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(ifs.IVNic)
		return true
	})
	return result
}

// shutdownConnection closes the connection with the given UUID from both internal and external maps.
func (this *Connections) shutdownConnection(uuid string) {
	this.logger.Debug("Shutting down connection ", uuid)
	conn, ok := this.internal.Load(uuid)
	if ok {
		conn.(ifs.IVNic).Shutdown()
	}
	conn, ok = this.externalVnet.Load(uuid)
	if ok {
		conn.(ifs.IVNic).Shutdown()
	}
}

// allDownConnections returns a map of UUIDs for all connections that are not currently running.
func (this *Connections) allDownConnections() map[string]bool {
	result := make(map[string]bool)
	this.internal.Range(func(key, value interface{}) bool {
		if !value.(ifs.IVNic).Running() {
			result[key.(string)] = true
		}
		return true
	})
	this.externalVnet.Range(func(key, value interface{}) bool {
		if !value.(ifs.IVNic).Running() {
			result[key.(string)] = true
		}
		return true
	})
	return result
}

// Routes returns a map of internal connection UUIDs to the VNet UUID for routing purposes.
func (this *Connections) Routes() map[string]string {
	routes := make(map[string]string)
	this.internal.Range(func(key, value interface{}) bool {
		routes[key.(string)] = this.vnetUuid
		return true
	})
	/*
		for k, v := range this.routes {
			routes[k] = v
		}*/
	return routes
}
