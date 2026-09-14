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

package t_service

import (
	"sync"
)

func (this *TestServiceBase) Reset() {
	this.postNumber.Add(this.postNumber.Load() * -1)
	this.putNumber.Add(this.putNumber.Load() * -1)
	this.patchNumber.Add(this.patchNumber.Load() * -1)
	this.getNumber.Add(this.getNumber.Load() * -1)
	this.deleteNumber.Add(this.deleteNumber.Load() * -1)
	this.errorMode = false
}

func (this *TestServiceBase) Name() string {
	return this.name
}

func (this *TestServiceBase) ErrorMode() bool {
	return this.errorMode
}

func (this *TestServiceBase) SetErrorMode(b bool) {
	this.errorMode = b
}

func (this *TestServiceBase) PostN() int {
	return int(this.postNumber.Load())
}

func (this *TestServiceBase) PostNReplica() *sync.Map {
	return this.postReplica
}

func (this *TestServiceBase) PutN() int {
	return int(this.putNumber.Load())
}

func (this *TestServiceBase) PatchN() int {
	return int(this.patchNumber.Load())
}

func (this *TestServiceBase) GetN() int {
	return int(this.getNumber.Load())
}

func (this *TestServiceBase) GetNReplica() *sync.Map {
	return this.getReplica
}

func (this *TestServiceBase) DeleteN() int {
	return int(this.deleteNumber.Load())
}

func (this *TestServiceBase) FailedN() int {
	return int(this.failedNumber.Load())
}
