/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package nets provides low-level network protocol operations for the Layer 8 system.
// It handles connection establishment, handshake protocol execution, and
// encrypted data transmission over TCP connections.
package nets

import (
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8services"
	"github.com/saichler/l8types/go/types/l8sysconfig"
	"google.golang.org/protobuf/proto"

	"net"
)

// ExecuteProtocol performs the connection handshake between two Layer 8 nodes.
// This exchanges UUIDs, aliases, configuration flags, and service registrations.
// All data is encrypted during transmission for security.
// The protocol flow:
//  1. Exchange local/remote UUIDs
//  2. Exchange force-external flags
//  3. Exchange aliases
//  4. Exchange service registrations
//  5. Exchange remote VNet information
func ExecuteProtocol(conn net.Conn, config *l8sysconfig.L8SysConfig, security ifs.ISecurityProvider) error {
	err := WriteEncrypted(conn, []byte(config.LocalUuid), config, security)
	if err != nil {
		conn.Close()
		return err
	}

	config.RemoteUuid, err = ReadEncrypted(conn, config, security)
	if err != nil {
		conn.Close()
		return err
	}

	forceExternal := "false"
	if config.ForceExternal {
		forceExternal = "true"
	}

	err = WriteEncrypted(conn, []byte(forceExternal), config, security)
	if err != nil {
		conn.Close()
		return err
	}

	forceExternal, err = ReadEncrypted(conn, config, security)
	if err != nil {
		conn.Close()
		return err
	}
	if forceExternal == "true" {
		config.ForceExternal = true
	}

	err = WriteEncrypted(conn, []byte(config.LocalAlias), config, security)
	if err != nil {
		conn.Close()
		return err
	}

	remoteAlias, err := ReadEncrypted(conn, config, security)
	if err != nil {
		conn.Close()
		return err
	}
	config.RemoteAlias = remoteAlias

	err = WriteEncrypted(conn, ServicesToBytes(config.Services), config, security)
	if err != nil {
		conn.Close()
		return err
	}

	services, err := ReadEncryptedBytes(conn, config, security)
	if err != nil {
		conn.Close()
		return err
	}
	config.Services = BytesToServices(services)

	err = WriteEncrypted(conn, []byte(config.RemoteVnet), config, security)
	if err != nil {
		conn.Close()
		return err
	}

	remoteVnet, err := ReadEncrypted(conn, config, security)
	if err != nil {
		conn.Close()
		return err
	}
	if config.RemoteVnet == "" {
		config.RemoteVnet = remoteVnet
	}
	return nil
}

// ServicesToBytes serializes a services registry to protobuf bytes.
func ServicesToBytes(services *l8services.L8Services) []byte {
	data, err := proto.Marshal(services)
	if err != nil {
		return []byte{}
	}
	return data
}

// BytesToServices deserializes protobuf bytes to a services registry.
func BytesToServices(data []byte) *l8services.L8Services {
	services := &l8services.L8Services{}
	err := proto.Unmarshal(data, services)
	if err != nil {
		return nil
	}
	return services
}
