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
	vnic2 "github.com/saichler/l8bus/go/overlay/vnic"
	"github.com/saichler/l8types/go/types/l8sysconfig"
	resources2 "github.com/saichler/l8utils/go/utils/resources"
)

// ConnectNetworks establishes a connection to a remote VNet at the specified host and port.
// It creates a new VNic for the connection, validates security credentials, and registers
// the connection for health monitoring and routing.
func (this *VNet) ConnectNetworks(host string, destPort uint32) error {
	sec := this.resources.Security()
	// Dial the destination and validate the secret and key
	conn, err := sec.CanDial(host, destPort)
	if err != nil {
		return err
	}

	config := &l8sysconfig.L8SysConfig{MaxDataSize: resources2.DEFAULT_MAX_DATA_SIZE,
		RxQueueSize:   resources2.DEFAULT_QUEUE_SIZE,
		TxQueueSize:   resources2.DEFAULT_QUEUE_SIZE,
		VnetPort:      destPort,
		LocalUuid:     this.resources.SysConfig().LocalUuid,
		Services:      this.resources.SysConfig().Services,
		ForceExternal: true,
		LocalAlias:    this.resources.SysConfig().LocalAlias,
	}

	resources := resources2.NewResources(this.resources.Logger())
	resources.Copy(this.resources)

	resources.Set(config)
	resources.Set(this)

	vnic := vnic2.NewVirtualNetworkInterface(resources, conn)

	err = sec.ValidateConnection(conn, config)
	if err != nil {
		return err
	}

	vnic.Start()
	this.addHealthForVNic(vnic.Resources().SysConfig())
	this.notifyNewVNic(vnic)
	return nil
}
