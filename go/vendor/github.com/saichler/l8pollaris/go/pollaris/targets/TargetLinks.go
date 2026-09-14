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

package targets

// TargetLinks defines the interface for routing targets to various services.
// Implementations provide the mapping from a links ID to the appropriate
// service name and service area for each type of processing pipeline.
// This abstraction allows different target types to be routed to different
// collector, parser, cache, and persistence implementations.
type TargetLinks interface {
	// Collector returns the service name and area for the data collector
	// that should poll this target based on its links ID.
	Collector(string) (string, byte)
	// Parser returns the service name and area for the data parser
	// that should process collected data for this target.
	Parser(string) (string, byte)
	// Cache returns the service name and area for the cache service
	// that should store parsed data for this target.
	Cache(string) (string, byte)
	// Persist returns the service name and area for the persistence service
	// that should store data long-term for this target.
	Persist(string) (string, byte)
	// The model name
	Model(string) string
}
