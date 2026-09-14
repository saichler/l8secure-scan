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
	"net"
	stdstrings "strings"
	"time"

	"github.com/saichler/l8utils/go/utils/ipsegment"
)

// DNSDiscovery handles peer VNet discovery using DNS hostname resolution.
// When pointed at a K8s headless Service name, net.LookupHost returns all pod IPs,
// enabling automatic peer connection without UDP broadcast.
type DNSDiscovery struct {
	vnet       *VNet
	dnsName    string
	discovered map[string]bool
}

// NewDNSDiscovery creates a new DNSDiscovery for the given VNet and DNS hostname.
func NewDNSDiscovery(vnet *VNet, dnsName string) *DNSDiscovery {
	dd := &DNSDiscovery{}
	dd.vnet = vnet
	dd.dnsName = dnsName
	dd.discovered = make(map[string]bool)
	return dd
}

// Discover starts the DNS-based discovery loop.
func (this *DNSDiscovery) Discover() {
	this.vnet.resources.Logger().Debug("DNS Discovery: starting lookup loop for ", this.dnsName)
	this.resolve()
	time.Sleep(time.Second * 10)
	this.resolve()
	for this.vnet.running {
		time.Sleep(time.Minute)
		this.resolve()
	}
}

func (this *DNSDiscovery) resolve() {
	ips, err := net.LookupHost(this.dnsName)
	if err != nil {
		this.vnet.resources.Logger().Error("DNS Discovery: lookup failed for ", this.dnsName, ": ", err.Error())
		return
	}

	vnetPort := this.vnet.resources.SysConfig().VnetPort
	for _, ip := range ips {
		if ip == ipsegment.MachineIP || ip == "127.0.0.1" {
			continue
		}

		_, alreadyKnown := this.discovered[ip]

		if stdstrings.Compare(ip, ipsegment.MachineIP) == -1 && !alreadyKnown && !this.vnet.switchTable.conns.isConnected(ip) {
			this.vnet.resources.Logger().Debug("DNS Discovery: connecting to peer at ", ip)
			err = this.vnet.ConnectNetworks(ip, vnetPort)
			if err != nil {
				this.vnet.resources.Logger().Error("DNS Discovery: ", err.Error())
			}
		}

		this.discovered[ip] = true
	}
}
