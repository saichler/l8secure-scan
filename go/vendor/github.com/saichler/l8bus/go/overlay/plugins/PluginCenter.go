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

package plugins

import (
	"crypto/md5"
	"encoding/base64"
	"errors"
	"os"
	"plugin"
	"sync"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8web"
	"github.com/saichler/l8utils/go/utils/strings"
)

// loadedPlugins caches loaded plugins by their MD5 hash to prevent duplicate loading.
var loadedPlugins = make(map[string]*plugin.Plugin)

// mtx protects concurrent access to the loadedPlugins map.
var mtx = &sync.Mutex{}

// loadPluginFile loads a plugin from base64-encoded data. It caches plugins by
// MD5 hash to avoid reloading identical plugins. The plugin is written to a
// temporary .so file and loaded using Go's plugin package.
func loadPluginFile(p *l8web.L8Plugin) (*plugin.Plugin, error) {

	md5 := md5.New()
	md5Hash := base64.StdEncoding.EncodeToString(md5.Sum([]byte(p.Data)))
	mtx.Lock()
	defer mtx.Unlock()
	pluginFile, ok := loadedPlugins[md5Hash]
	if ok {
		return pluginFile, nil
	}

	data, err := base64.StdEncoding.DecodeString(p.Data)
	if err != nil {
		return nil, err
	}
	name := strings.New(ifs.NewUuid(), ".so").String()
	err = os.WriteFile(name, data, 0777)
	if err != nil {
		return nil, err
	}
	defer os.Remove(name)

	pluginFile, err = plugin.Open(name)
	if err != nil {
		return nil, errors.New(strings.New("failed to load plugin #1 ", err.Error()).String())
	}

	loadedPlugins[md5Hash] = pluginFile

	return pluginFile, nil
}

// LoadPlugin loads and installs a plugin from the given L8Plugin data.
// The plugin must export a "Plugin" symbol implementing the IPlugin interface.
// After loading, the plugin's Install method is called with the provided VNic.
func LoadPlugin(p *l8web.L8Plugin, vnic ifs.IVNic) error {
	pluginFile, err := loadPluginFile(p)
	if err != nil {
		return err
	}

	plg, err := pluginFile.Lookup("Plugin")
	if err != nil {
		return errors.New("failed to load plugin #2")
	}
	if plg == nil {
		return errors.New("failed to load plugin #3")
	}
	pluginInterface := *plg.(*ifs.IPlugin)
	err = pluginInterface.Install(vnic)
	if err != nil {
		return err
	}
	return err
}
