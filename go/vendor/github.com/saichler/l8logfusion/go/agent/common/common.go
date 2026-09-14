/*
 * © 2025 Sharon Aicler (saichler@gmail.com)
 *
 * Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package common provides shared constants and utility functions used across
// the L8LogFusion log aggregation system. It defines service identifiers and
// common file operations for the log collection infrastructure.
package common

// Service registration constants for the L8LogFusion log service.
// These values are used to identify and route messages within the Layer 8 virtual network.
const (
	// LogServiceName is the unique identifier for the log aggregation service
	// used for service discovery and message routing in the L8Bus overlay network.
	LogServiceName = "logs"

	// LogServiceArea defines the service area code for the log service.
	// This byte value (87) is used to partition services within the virtual network
	// for efficient message routing and load balancing.
	LogServiceArea = byte(87)
)
