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

// Read.go provides network read operations for the Layer 8 protocol.
// All reads use a length-prefixed format: 8 bytes (int64) size followed by data.

package nets

import (
	"errors"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8sysconfig"

	"net"
	"time"
)

// Read data from socket
func Read(conn net.Conn, config *l8sysconfig.L8SysConfig) ([]byte, error) {
	// If the connection is nil, return an error
	if conn == nil {
		return nil, errors.New("no Connection Available")
	}
	// If the config is nil, error
	if config == nil {
		return nil, errors.New("no Config Available")
	}
	// read 8 bytes, e.g. long, hinting of the size of the byte array
	sizebytes, err := ReadSize(8, conn, config)
	if sizebytes == nil || err != nil {
		return nil, err
	}
	// Translate the 8 byte array into int64
	size := ifs.Bytes2Long(sizebytes)
	// If the size is larger than the MAX Data Size, return an error
	// this is to protect against overflowing the buffers
	// When data to send is > the max data size, one needs to split the data into chunks at a higher level
	if uint64(size) > config.MaxDataSize {
		return nil, errors.New("Max Size Exceeded!")
	}
	// Read the bunch of bytes according to the size from the socket
	data, err := ReadSize(int(size), conn, config)
	return data, err
}

// ReadSize reads exactly 'size' bytes from the connection, handling partial reads.
// Will retry reads until all bytes are received or an error occurs.
func ReadSize(size int, conn net.Conn, config *l8sysconfig.L8SysConfig) ([]byte, error) {
	data := make([]byte, size)
	n, e := conn.Read(data)
	if e != nil {
		return nil, errors.New("Failed to read data size:" + e.Error())
	}

	if n < size {
		if n == 0 {
			time.Sleep(time.Second)
		}
		data = data[0:n]
		left, e := ReadSize(size-n, conn, config)
		if e != nil {
			return nil, errors.New("Failed to read packet size:" + e.Error())
		}
		data = append(data, left...)
	}
	return data, nil
}

// ReadEncryptedBytes reads encrypted data from the connection and decrypts it.
// Returns the decrypted data as bytes.
func ReadEncryptedBytes(conn net.Conn, config *l8sysconfig.L8SysConfig,
	securityProvider ifs.ISecurityProvider) ([]byte, error) {
	inData, err := Read(conn, config)
	if err != nil {
		conn.Close()
		return []byte{}, err
	}

	decData, err := securityProvider.Decrypt(string(inData))
	if err != nil {
		conn.Close()
		return []byte{}, err
	}
	return decData, nil
}

// ReadEncrypted reads encrypted data from the connection and returns it as a string.
func ReadEncrypted(conn net.Conn, config *l8sysconfig.L8SysConfig,
	securityProvider ifs.ISecurityProvider) (string, error) {
	data, err := ReadEncryptedBytes(conn, config, securityProvider)
	return string(data), err
}
