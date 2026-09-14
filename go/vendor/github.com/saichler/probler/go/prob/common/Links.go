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

package common

const (
	Collector_Service_Name = "Coll"
	Collector_Service_Area = byte(0)

	AdControl_Service_Name = "AdCon"
	AdControl_Service_Area = byte(1)

	NetworkDevice_Links_ID      = "NetDev"
	NetDev_Cache_Service_Name   = "NCache"
	NetDev_Cache_Service_Area   = byte(0)
	NetDev_Persist_Service_Name = "NPersist"
	NetDev_Persist_Service_Area = byte(0)
	NetDev_Parser_Service_Name  = "NPars"
	NetDev_Parser_Service_Area  = byte(0)
	NetDev_Model_Name           = "networkdevice"

	K8s_Persist_Service_Name = "KPersist"
	K8s_Persist_Service_Area = byte(1)
	K8s_Parser_Service_Name  = "KPars"

	GPU_Links_ID             = "GPU"
	GPU_Cache_Service_Name   = "GCache"
	GPU_Cache_Service_Area   = byte(2)
	GPU_Persist_Service_Name = "GPersist"
	GPU_Persist_Service_Area = byte(2)
	GPU_Parser_Service_Name  = "GPars"
	GPU_Parser_Service_Area  = byte(2)
	GPU_Model_Name           = "gpudevice"

	K8sC_Links_ID = "K8sC"
)

type Links struct{}

func (this *Links) Collector(linkid string) (string, byte) {
	if linkid == K8sC_Links_ID {
		return AdControl_Service_Name, AdControl_Service_Area
	}
	return Collector_Service_Name, Collector_Service_Area
}

func (this *Links) Parser(linkid string) (string, byte) {
	if name, area, ok := k8sParser(linkid); ok {
		return name, area
	}
	if linkid == K8sC_Links_ID {
		e := k8sLinkMap[K8sClust_Links_ID]
		return e.ParserName, e.ParserArea
	}
	switch linkid {
	case NetworkDevice_Links_ID:
		return NetDev_Parser_Service_Name, NetDev_Parser_Service_Area
	case GPU_Links_ID:
		return GPU_Parser_Service_Name, GPU_Parser_Service_Area
	}
	return "", 0
}

func (this *Links) Cache(linkid string) (string, byte) {
	if name, area, ok := k8sCache(linkid); ok {
		return name, area
	}
	if linkid == K8sC_Links_ID {
		e, _ := k8sLinkMap[K8sClust_Links_ID]
		return e.CacheName, e.CacheArea
	}
	switch linkid {
	case NetworkDevice_Links_ID:
		return NetDev_Cache_Service_Name, NetDev_Cache_Service_Area
	case GPU_Links_ID:
		return GPU_Cache_Service_Name, GPU_Cache_Service_Area
	}
	return "", 0
}

func (this *Links) Persist(linkid string) (string, byte) {
	if name, area, ok := k8sPersist(linkid); ok {
		return name, area
	}
	if linkid == K8sC_Links_ID {
		e, _ := k8sLinkMap[K8sClust_Links_ID]
		return e.PersistName, e.PersistArea
	}
	switch linkid {
	case NetworkDevice_Links_ID:
		return NetDev_Persist_Service_Name, NetDev_Persist_Service_Area
	case GPU_Links_ID:
		return GPU_Persist_Service_Name, GPU_Persist_Service_Area
	}
	return "", 0
}

func (this *Links) Model(linkid string) string {
	if name, ok := k8sModel(linkid); ok {
		return name
	}
	if linkid == K8sC_Links_ID {
		e, _ := k8sLinkMap[K8sClust_Links_ID]
		return e.ModelName
	}
	switch linkid {
	case NetworkDevice_Links_ID:
		return NetDev_Model_Name
	case GPU_Links_ID:
		return GPU_Model_Name
	}
	return ""
}
