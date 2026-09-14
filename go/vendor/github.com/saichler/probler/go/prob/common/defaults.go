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

import (
	"fmt"
	"github.com/saichler/l8pollaris/go/pollaris/targets"
	"github.com/saichler/l8reflect/go/reflect/introspecting"
	"github.com/saichler/l8services/go/services/manager"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/sec"
	"github.com/saichler/l8utils/go/utils/logger"
	"github.com/saichler/l8utils/go/utils/registry"
	"github.com/saichler/l8utils/go/utils/resources"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

/*
const (
	PROBLER_VNET    = 26000
	LOGS_VNET       = 27000
	PREFIX          = "/probler/"
	DB_CREDS        = "postgres"
	DB_TARGETS_NAME = "problerdb"
	DB_ALARMS_NAME  = "probleralarms"
)*/

func init() {
	targets.Links = &Links{}
}

func CreateResources(alias string) ifs.IResources {

	log := logger.NewLoggerImpl(&logger.FmtLogMethod{})
	log.SetLogLevel(ifs.Info_Level)
	res := resources.NewResources(log)

	res.Set(registry.NewRegistry())

	sec, err := sec.LoadSecurityProvider(res)
	if err != nil {
		fmt.Println("Failed to load security provider", err)
	} else {
		res.Set(sec)
	}

	if res.SysConfig().LogConfig != nil && res.SysConfig().LogConfig.LogDirectory != "" {
		logger.SetLogToFile(res.SysConfig().LogConfig.LogDirectory, alias)
	}

	res.SysConfig().LocalAlias = alias + "-" + strconv.Itoa(int(res.SysConfig().VnetPort))
	res.SysConfig().KeepAliveIntervalSeconds = 30

	res.Set(introspecting.NewIntrospect(res.Registry()))
	res.Set(manager.NewServices(res))

	return res
}

func WaitForSignal(resources ifs.IResources) {
	resources.Logger().Info("Waiting for os signal...")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	resources.Logger().Info("End signal received! ", sig)
}
