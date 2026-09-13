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
package common

import (
	"fmt"
	"strings"
)

// L8UserJSON builds a protojson-compatible JSON payload for creating an L8User
// via the Security API (service area 73). The field names match the protobuf
// JSON serialization produced by protojson.Marshal on an L8User instance.
//
// Parameters:
//   - userId: unique user identifier (typically the entity's primary key or email)
//   - fullName: display name
//   - email: login email address
//   - password: initial plaintext password (server will salt and hash it)
//   - portal: post-login redirect path (e.g., "client-app.html", "member/app.html")
//   - roles: map of role names to enabled (e.g., {"client": true})
func L8UserJSON(userId, fullName, email, password, portal string, roles map[string]bool) string {
	var roleEntries []string
	for k, v := range roles {
		roleEntries = append(roleEntries, fmt.Sprintf(`"%s":%v`, k, v))
	}
	rolesJSON := "{" + strings.Join(roleEntries, ",") + "}"

	return fmt.Sprintf(`{
  "fullName": "%s",
  "userId": "%s",
  "password": {"hash": "%s"},
  "roles": %s,
  "email": "%s",
  "accountStatus": "ACCOUNT_STATUS_ACTIVE",
  "portal": "%s"
}`, fullName, userId, password, rolesJSON, email, portal)
}
