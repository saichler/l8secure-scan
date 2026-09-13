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
	"database/sql"
	"encoding/base64"
	"fmt"
	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8bus/go/overlay/vnic"
	"github.com/saichler/l8logfusion/go/types/l8logf"
	"github.com/saichler/l8reflect/go/reflect/introspecting"
	"github.com/saichler/l8services/go/services/csvexport"
	"github.com/saichler/l8services/go/services/dataimport"
	"github.com/saichler/l8services/go/services/filestore"
	"github.com/saichler/l8services/go/services/manager"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/sec"
	"github.com/saichler/l8utils/go/utils/ipsegment"
	"github.com/saichler/l8utils/go/utils/logger"
	"github.com/saichler/l8utils/go/utils/registry"
	"github.com/saichler/l8utils/go/utils/resources"
	"github.com/saichler/l8web/go/web/server"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var dbInstance *sql.DB
var dbMtx = &sync.Mutex{}

// CreateResources creates the standard Layer 8 resources with configurable parameters.
func CreateResources(alias string, logVnet bool) ifs.IResources {
	log := logger.NewLoggerImpl(&logger.FmtLogMethod{})
	log.SetLogLevel(ifs.Info_Level)
	res := resources.NewResources(log)

	res.Set(registry.NewRegistry())

	sec, _ := sec.LoadSecurityProvider(res)
	if sec != nil {
		res.Set(sec)
	}

	res.SysConfig().LocalAlias = alias

	if logVnet {
		res.SysConfig().VnetPort = res.SysConfig().LogConfig.VnetPort
	}

	if res.SysConfig().LogConfig != nil && res.SysConfig().LogConfig.LogDirectory != "" {
		logger.SetLogToFile(res.SysConfig().LogConfig.LogDirectory, alias)
	}

	res.Set(introspecting.NewIntrospect(res.Registry()))
	res.Set(manager.NewServices(res))

	return res
}

// WaitForSignal blocks until the process receives SIGINT or SIGTERM.
func WaitForSignal(resources ifs.IResources) {
	resources.Logger().Info("Waiting for os signal...")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	resources.Logger().Info("End signal received! ", sig)
}

// OpenDBConection establishes a connection to the PostgreSQL database.
// It uses localhost (127.0.0.1) on port 5432 with SSL disabled.
// Panics if the connection cannot be established or ping fails.
func OpenDBConection(dbname, user, pass, port string) *sql.DB {
	dbMtx.Lock()
	defer dbMtx.Unlock()
	if dbInstance != nil {
		return dbInstance
	}
	p, _ := strconv.Atoi(port)
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		ipsegment.MachineIP, p, user, pass, dbname)
	db, err := sql.Open("postgres", psqlInfo)

	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(fmt.Errorf("failed to connect to database: %w", err))
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	dbInstance = db
	return dbInstance
}

func CreateWebServer(alias string, registerTypes func(r ifs.IResources)) ifs.IWebServer {
	nic1, nic2 := createWebVnics(alias, registerTypes)
	server.UpdateLoginJsonPrefix(nic1.Resources().SysConfig().WebConfig.EndPointPrefix)

	domain, private, _ := nic1.Resources().Certificate()
	if domain == "" || private == "" {
		d, p, _ := sec.CreateCertBundle()
		domainBytes, _ := base64.StdEncoding.DecodeString(d)
		privateBytes, _ := base64.StdEncoding.DecodeString(p)
		domain = string(domainBytes)
		private = string(privateBytes)
	}

	serverConfig := &server.RestServerConfig{
		Host:           ipsegment.MachineIP,
		Port:           int(nic1.Resources().SysConfig().WebConfig.WebPort),
		Authentication: true,
		CertDomain:     domain,
		CertPrivate:    private,
		Prefix:         nic1.Resources().WebPrefix(),
	}

	svr, err := server.NewRestServer(serverConfig)
	if err != nil {
		panic(err)
	}

	csvexport.Activate(nic1)
	filestore.Activate(nic1)
	dataimport.Activate(nic1)

	hs, ok := nic1.Resources().Services().ServiceHandler(health.ServiceName, 0)
	if ok {
		ws := hs.WebService()
		svr.RegisterWebService(ws, nic1)
	}

	//Activate the webpoints service
	sla := ifs.NewServiceLevelAgreement(&server.WebService{}, ifs.WebService, 0, false, nil)
	if nic2 == nil {
		sla.SetArgs(svr)
	} else {
		sla.SetArgs(svr, nic2)
	}
	nic1.Resources().Services().Activate(sla, nic1)
	nic1.Resources().Logger().Info("Web Server Started!")

	return svr
}

func createWebVnics(alias string, registerTypes func(r ifs.IResources)) (ifs.IVNic, ifs.IVNic) {
	nic1 := CreateVnic(alias+"-f", false, registerTypes)
	var nic2 ifs.IVNic
	if nic1.Resources().SysConfig().LogConfig != nil && nic1.Resources().SysConfig().LogConfig.LogDirectory != "" {
		nic2 = CreateVnic(alias+"-t", true, registerTypes)
	}
	return nic1, nic2
}

func CreateVnic(alias string, logs bool, registerTypes func(r ifs.IResources)) ifs.IVNic {
	res := CreateResources(alias, logs)
	res.Introspector().Decorators().AddPrimaryKeyDecorator(&l8logf.L8File{}, "Path", "Name")
	if registerTypes != nil {
		registerTypes(res)
	}
	nic := vnic.NewVirtualNetworkInterface(res, nil)
	nic.Start()
	nic.WaitForConnection()
	return nic
}
