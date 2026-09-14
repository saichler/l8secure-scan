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

	"github.com/saichler/l8bus/go/overlay/protocol"
	"github.com/saichler/l8utils/go/utils/ipsegment"
	"github.com/saichler/l8utils/go/utils/strings"
)

// Discovery handles peer VNet discovery using UDP broadcast.
// It enables automatic detection and connection to other VNet switches on the local network.
type Discovery struct {
	vnet         *VNet
	conn         *net.UDPConn
	discovered   map[string]bool
	dnsDiscovery *DNSDiscovery
}

// NewDiscovery creates a new Discovery instance for the given VNet.
func NewDiscovery(vnet *VNet) *Discovery {
	ds := &Discovery{}
	ds.vnet = vnet
	ds.discovered = make(map[string]bool)
	dnsName := vnet.resources.SysConfig().DiscoveryDnsName
	if dnsName != "" {
		ds.dnsDiscovery = NewDNSDiscovery(vnet, dnsName)
	}
	return ds
}

// Discover starts the discovery process by listening for UDP broadcasts
// and initiating connections to discovered peer VNets.
func (this *Discovery) Discover() {
	if this.dnsDiscovery != nil {
		go this.dnsDiscovery.Discover()
	}

	if !protocol.Discovery_Enabled {
		this.vnet.resources.Logger().Debug("Discovery is disabled, machine IP is ", ipsegment.MachineIP)
		return
	}
	addr, err := net.ResolveUDPAddr("udp", strings.New(":", int(this.vnet.resources.SysConfig().VnetPort-2)).String())
	if err != nil {
		this.vnet.resources.Logger().Error("Discovery: ", err.Error())
		return
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		this.vnet.resources.Logger().Error("Discovery: ", err.Error())
		return
	}
	this.conn = conn
	go this.discoveryRx()
	go this.Broadcast()
}

func (this *Discovery) discoveryRx() {
	this.vnet.resources.Logger().Debug("Listening for discovery broadcast")
	packet := []byte{0, 0, 0}
	defer this.conn.Close()

	for this.vnet.running {
		n, addr, err := this.conn.ReadFromUDP(packet)
		ip := addr.IP.String()
		this.vnet.resources.Logger().Debug("Recevied discovery broadcast from ", ip, " size ", n)
		if !this.vnet.running {
			break
		}
		if err != nil {
			this.vnet.resources.Logger().Error(err.Error())
			break
		}
		if n == 3 {
			if ip != ipsegment.MachineIP && ip != "127.0.0.1" {
				_, ok := this.discovered[ip]
				if stdstrings.Compare(ip, ipsegment.MachineIP) == -1 && !ok && !this.vnet.switchTable.conns.isConnected(ip) {
					this.vnet.resources.Logger().Debug("Trying to connect to peer at ", ip)
					err = this.vnet.ConnectNetworks(ip, this.vnet.resources.SysConfig().VnetPort)
					if err != nil {
						this.vnet.resources.Logger().Error("Discovery: ", err.Error())
					}
				}
				this.discovered[ip] = true
			}
		}
	}
}

// Broadcast sends periodic UDP discovery broadcasts to announce this VNet's presence.
func (this *Discovery) Broadcast() {
	this.vnet.resources.Logger().Debug("Sending discovery broadcast")
	addr, err := net.ResolveUDPAddr("udp", strings.New("255.255.255.255:", int(this.vnet.resources.SysConfig().VnetPort-2)).String())
	if err != nil {
		this.vnet.resources.Logger().Error("Failed to resolve broadcast:", err.Error())
		return
	}
	this.conn.WriteToUDP([]byte{1, 2, 3}, addr)
	time.Sleep(time.Second * 10)
	this.vnet.resources.Logger().Debug("Sending discovery broadcast")
	this.conn.WriteToUDP([]byte{1, 2, 3}, addr)
	for this.vnet.running {
		time.Sleep(time.Minute)
		this.vnet.resources.Logger().Debug("Sending discovery broadcast")
		this.conn.WriteToUDP([]byte{1, 2, 3}, addr)
	}
}
