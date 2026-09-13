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

package cache

func (this *internalCache) putUnique(pk, uk string) {
	if uk == "" {
		return
	}
	// Clean up old unique key mapping if it exists and is different
	if oldUk, exists := this.PrimaryToUnique[pk]; exists && oldUk != uk {
		delete(this.UniqueToPrimary, oldUk)
	}
	this.hasExtraKeys = true
	this.UniqueToPrimary[uk] = pk
	this.PrimaryToUnique[pk] = uk
}

func (this *internalCache) deleteUnique(pk, uk string) {
	// If uk not provided, look it up before deleting
	if uk == "" {
		uk = this.PrimaryToUnique[pk]
	}
	delete(this.UniqueToPrimary, uk)
	delete(this.PrimaryToUnique, pk)
}
