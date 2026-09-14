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

package shared

import (
	"fmt"
	"github.com/saichler/l8reflect/go/reflect/introspecting"
	"github.com/saichler/l8services/go/services/manager"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/sec"
	"github.com/saichler/l8utils/go/utils/logger"
	"github.com/saichler/l8utils/go/utils/registry"
	"github.com/saichler/l8utils/go/utils/resources"
)

func ResourcesOf(alias string, vnetPort, keepAlive uint32, others ...ifs.IResources) ifs.IResources {

	log := logger.NewLoggerImpl(&logger.FmtLogMethod{})
	log.SetLogLevel(ifs.Error_Level)
	res := resources.NewResources(log)

	res.Set(registry.NewRegistry())

	if others != nil {
		for _, other := range others {
			res.Set(other.Security())
		}
	} else {
		sec, err := sec.LoadSecurityProvider(res)
		if err != nil {
			fmt.Println("*** Failed to load security provider! ***")
			fmt.Println("*** Using Shallow Security Provider!  ***")
		}
		res.Set(sec)
	}
	res.Set(res.Security().NewSystemConfig())

	res.SysConfig().VnetPort = vnetPort
	res.SysConfig().LocalAlias = alias
	res.SysConfig().KeepAliveIntervalSeconds = int64(keepAlive)

	if res.SysConfig().LogConfig != nil && res.SysConfig().LogConfig.LogDirectory != "" {
		logger.SetLogToFile(res.SysConfig().LogConfig.LogDirectory, alias)
	}

	res.Set(introspecting.NewIntrospect(res.Registry()))
	res.Set(manager.NewServices(res))

	return res
}
