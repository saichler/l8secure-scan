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

// Write.go provides network write operations for the Layer 8 protocol.
// All writes use a length-prefixed format: 8 bytes (int64) size followed by data.

package nets

import (
	"errors"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8sysconfig"

	"net"
)

// Write data to socket
func Write(data []byte, conn net.Conn, config *l8sysconfig.L8SysConfig) error {
	// If the connection is nil, return an error
	if conn == nil {
		return errors.New("no Connection Available")
	}
	// If the config is nil, error
	if config == nil {
		return errors.New("no Config Available")
	}
	if data == nil {
		return errors.New("no Data Available")
	}
	// Error is the data is too big
	if len(data) > int(config.MaxDataSize) {
		return errors.New("data is larger than MAX size allowed")
	}
	// Write the size of the data
	_, e := conn.Write(ifs.Long2Bytes(int64(len(data))))
	if e != nil {
		return e
	}
	// Write the actual data
	_, e = conn.Write(data)
	return e
}

// WriteEncrypted encrypts data and writes it to the connection.
func WriteEncrypted(conn net.Conn, data []byte, config *l8sysconfig.L8SysConfig,
	securityProvider ifs.ISecurityProvider) error {
	encData, err := securityProvider.Encrypt(data)
	if err != nil {
		return err
	}
	err = Write([]byte(encData), conn, config)
	if err != nil {
		return err
	}
	return nil
}
