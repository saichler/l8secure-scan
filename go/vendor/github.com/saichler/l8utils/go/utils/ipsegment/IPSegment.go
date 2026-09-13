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

// Package ipsegment provides network interface and IP address detection utilities.
// It determines whether IP addresses are local or external and identifies the machine's
// primary network IP address across different operating systems.
//
// Key features:
//   - Local vs external IP classification
//   - Automatic machine IP detection on initialization
//   - Cross-platform support (Linux, macOS, Windows, Android)
//   - Interface name to IP address mapping
//   - Subnet-based IP categorization
package ipsegment

import (
	"errors"
	"io"
	"net"
	"net/http"
	"regexp"
	"runtime"
	stdstrings "strings"
	"time"

	"github.com/saichler/l8utils/go/utils/strings"
)

// IpSegment is the global IP segment detector, initialized automatically on package load.
var IpSegment = newIpAddressSegment()

// MachineIP holds the detected external IP address of this machine.
var MachineIP = "127.0.0.1"

// IPSegment Let the switching know if the incoming ip belongs to this machine/vm or is it external machine/vm.
type IPSegment struct {
	ip2IfName    map[string]string
	subnet2Local map[string]bool
}

// Initialize
func newIpAddressSegment() *IPSegment {
	ias := &IPSegment{}
	lip, err := LocalIps()
	if err != nil {
		panic(err)
	}
	ias.ip2IfName = lip
	ias.initSegment()

	_, MachineIP, _ = ias.DetectOSAndExternalIP()

	return ias
}

// Initiate and destinguish all the interfaces if they are local or public
// @TODO - Find a more elegant way to determinate this, like a map
func (ias *IPSegment) initSegment() {
	ias.subnet2Local = make(map[string]bool)
	for ip, name := range ias.ip2IfName {
		if name == "lo" {
			ias.subnet2Local[Subnet(ip)] = true
		} else if name[0:3] == "eth" ||
			name[0:3] == "ens" ||
			name[0:3] == "en0" ||
			name[0:3] == "wlp" ||
			name[0:3] == "enp" {
			ias.subnet2Local[Subnet(ip)] = false
		} else {
			ias.subnet2Local[Subnet(ip)] = true
		}
	}
}

// Check if this ip's subnet is within the local subnet list
func (ias *IPSegment) IsLocal(ip string) bool {
	ip = IP(ip)
	if ip == MachineIP {
		return true
	}
	return ias.subnet2Local[Subnet(ip)]
}

// look for the subnet facing public networking, e.g. the ip on eth0 & etc.
// @TODO - Add support for multiple NICs
func (ias *IPSegment) ExternalSubnet() string {
	for subnet, isLocal := range ias.subnet2Local {
		if !isLocal {
			return subnet
		}
	}
	return ""
}

// substr the subnet from an ip
// @TODO - add support for ipv6
func Subnet(ip string) string {
	index2 := stdstrings.LastIndex(ip, ".")
	if index2 != -1 {
		return ip[0:index2]
	}
	return ip
}

func IP(ip string) string {
	index := stdstrings.Index(ip, "/")
	if index != -1 {
		return ip[0:index]
	}
	index = stdstrings.LastIndex(ip, ":")
	if index != -1 {
		return ip[0:index]
	}
	return ip
}

// Iterate over the machine interfaces and map the ip to the interface name
// LocalIps returns a map of IP addresses to their interface names for all local interfaces.
func LocalIps() (map[string]string, error) {
	if runtime.GOOS == "android" {
		return map[string]string{"127.0.0.1": "eth0"}, nil
	}

	netIfs, err := net.Interfaces()
	if err != nil {
		return nil, errors.New(strings.New("Could not fetch local interfaces: ", err.Error()).String())
	}
	result := make(map[string]string)
	for _, netIf := range netIfs {
		addrs, err := netIf.Addrs()
		if err != nil {
			//logs.Error("Failed to fetch addresses for net interface:", err.Error())
			continue
		}
		for _, addr := range addrs {
			addrString := addr.String()
			index := stdstrings.Index(addrString, "/")
			result[addrString[0:index]] = netIf.Name
		}
	}
	return result, nil
}

// OSInfo contains detected operating system information.
type OSInfo struct {
	Name    string
	Version string
	Arch    string
}

// DetectOSAndExternalIP detects the operating system and local network IP address
func (ias *IPSegment) DetectOSAndExternalIP() (OSInfo, string, error) {
	// Detect OS
	osInfo := ias.detectOS()

	// Detect local network IP (machine's IP on local network)
	externalIP, err := ias.detectExternalIP()
	if err != nil {
		return osInfo, "", err
	}

	return osInfo, externalIP, nil
}

// detectOS detects the current operating system with detailed information
func (ias *IPSegment) detectOS() OSInfo {
	osInfo := OSInfo{
		Name: runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	// Add more specific OS detection for different platforms
	switch runtime.GOOS {
	case "android":
		osInfo.Version = ias.getAndroidVersion()
	case "linux":
		osInfo.Version = ias.getLinuxVersion()
	case "darwin":
		osInfo.Version = ias.getMacVersion()
	case "windows":
		osInfo.Version = ias.getWindowsVersion()
	default:
		osInfo.Version = "unknown"
	}

	return osInfo
}

// detectExternalIP detects the machine's IP address on the local network
func (ias *IPSegment) detectExternalIP() (string, error) {
	// Method 1: Get the primary network interface IP (non-loopback, non-private to internet)
	localNetworkIP, err := ias.getLocalNetworkIP()
	if err == nil {
		return localNetworkIP, nil
	}

	// Method 2: Fallback to existing external subnet detection
	return ias.getExternalIPFromInterfaces()
}

// queryExternalIPService queries an external service to get the public IP
func (ias *IPSegment) queryExternalIPService(serviceURL string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(serviceURL)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("HTTP request failed")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Clean and validate IP address
	ip := stdstrings.TrimSpace(string(body))
	if ias.isValidIP(ip) {
		return ip, nil
	}

	return "", errors.New("invalid IP address received")
}

// getLocalNetworkIP gets the machine's primary IP address on the local network
func (ias *IPSegment) getLocalNetworkIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	var candidates []string

	for _, iface := range interfaces {
		// Skip loopback, down, or virtual interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Prioritize physical interfaces (ethernet, wifi)
		isPhysical := ias.isPhysicalInterface(iface.Name)

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Look for private network IPs (192.168.x.x, 10.x.x.x, 172.16-31.x.x)
			if ip != nil && !ip.IsLoopback() && ip.IsPrivate() && ip.To4() != nil {
				if isPhysical {
					// Prioritize physical interface IPs
					candidates = append([]string{ip.String()}, candidates...)
				} else {
					candidates = append(candidates, ip.String())
				}
			}
		}
	}

	if len(candidates) > 0 {
		return candidates[0], nil
	}

	return "", errors.New("no local network IP found")
}

// isPhysicalInterface checks if the interface is a physical network interface
func (ias *IPSegment) isPhysicalInterface(name string) bool {
	physicalPrefixes := []string{
		"eth", "ens", "enp", "en0", // Ethernet
		"wlp", "wlan", "wifi", "wl", // Wireless
	}

	for _, prefix := range physicalPrefixes {
		if stdstrings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// getExternalIPFromInterfaces attempts to get external IP from network interfaces
func (ias *IPSegment) getExternalIPFromInterfaces() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	var candidates []string

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil && !ip.IsLoopback() && !ip.IsPrivate() {
				candidates = append(candidates, ip.String())
			}
		}
	}

	if len(candidates) > 0 {
		return candidates[0], nil
	}

	// If no public IP found, return the best local IP
	for subnet, isLocal := range ias.subnet2Local {
		if !isLocal {
			return ias.reconstructIPFromSubnet(subnet)
		}
	}

	return MachineIP, nil
}

// isValidIP validates if a string is a valid IP address
func (ias *IPSegment) isValidIP(ip string) bool {
	// IPv4 regex
	ipv4Regex := regexp.MustCompile(`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`)
	// IPv6 regex (simplified)
	ipv6Regex := regexp.MustCompile(`^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`)

	return ipv4Regex.MatchString(ip) || ipv6Regex.MatchString(ip) || net.ParseIP(ip) != nil
}

// reconstructIPFromSubnet reconstructs a full IP from subnet (fallback method)
func (ias *IPSegment) reconstructIPFromSubnet(subnet string) (string, error) {
	for ip := range ias.ip2IfName {
		if stdstrings.HasPrefix(ip, subnet) {
			return IP(ip), nil
		}
	}
	return "", errors.New("no IP found for subnet")
}

// OS-specific version detection methods
func (ias *IPSegment) getAndroidVersion() string {
	// For Android, we can try to read system properties
	return "Android"
}

func (ias *IPSegment) getLinuxVersion() string {
	// Could read /etc/os-release or /proc/version for more details
	return "Linux"
}

func (ias *IPSegment) getMacVersion() string {
	// Could use system_profiler or sw_vers for more details
	return "macOS"
}

func (ias *IPSegment) getWindowsVersion() string {
	// Could use registry or WMI for more details
	return "Windows"
}
