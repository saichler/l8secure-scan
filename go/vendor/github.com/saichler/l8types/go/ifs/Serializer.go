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

package ifs

// SerializerMode defines the serialization format.
type SerializerMode int

const (
	BINARY SerializerMode = 1 // Binary/protobuf serialization
	JSON   SerializerMode = 2 // JSON serialization
	STRING SerializerMode = 3 // String/text serialization
)

// ISerializer defines the interface for object serialization and deserialization.
type ISerializer interface {
	// Mode returns the serialization mode.
	Mode() SerializerMode
	// Marshal converts an object to bytes.
	Marshal(interface{}, IResources) ([]byte, error)
	// Unmarshal converts bytes back to an object.
	Unmarshal([]byte, IResources) (interface{}, error)
}

// IStorage defines a key-value storage interface for persistent data.
type IStorage interface {
	// Put stores a value with the given key.
	Put(string, interface{}) error
	// Get retrieves a value by key.
	Get(string) (interface{}, error)
	// Delete removes a value by key and returns the deleted value.
	Delete(string) (interface{}, error)
	// Collect applies a filter function and returns matching key-value pairs.
	Collect(f func(interface{}) (bool, interface{})) map[string]interface{}
	// CacheEnabled returns true if in-memory caching is enabled.
	CacheEnabled() bool
}
