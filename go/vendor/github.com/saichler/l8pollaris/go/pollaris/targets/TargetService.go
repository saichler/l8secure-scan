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

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/saichler/l8orm/go/orm/persist"
	"github.com/saichler/l8orm/go/orm/plugins/postgres"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8web"
	"github.com/saichler/l8utils/go/utils/web"
	"strconv"
)

const (
	// ServiceName is the registered name of the Targets service in the service registry.
	ServiceName = "Targets"
	// ServiceArea defines the service area (partition) for the Targets service.
	// Area 91 is a dedicated area for target management operations.
	ServiceArea = byte(91)
)

// Links is the global TargetLinks implementation that routes targets to services.
// This must be set by the application before calling Activate.
var Links TargetLinks

// Activate initializes and registers the Targets service with the VNic.
// It establishes a PostgreSQL database connection, creates the ORM service,
// sets up lifecycle callbacks, and configures web service endpoints.
// After activation, it starts the InitTargets goroutine to restore target state.
// Parameters:
//   - creds: credential identifier for database authentication
//   - dbname: PostgreSQL database name
//   - vnic: the virtual network interface for service communication
func Activate(creds, dbname string, vnic ifs.IVNic) {
	realdb, user, pass, port, err := vnic.Resources().Security().Credential(creds, dbname, vnic.Resources())
	fmt.Println("Creds:", creds, "Realdb:", realdb, "User:", user, "Port:", port)
	if err != nil {
		panic(err)
	}
	db := openDBConection(realdb, user, pass, port)
	p := postgres.NewPostgres(db, vnic.Resources())

	callback := newTargetCallback(p)

	sla := ifs.NewServiceLevelAgreement(&persist.OrmService{}, ServiceName, ServiceArea, true, callback)
	sla.SetServiceItem(&l8tpollaris.L8PTarget{})
	sla.SetServiceItemList(&l8tpollaris.L8PTargetList{})
	sla.SetPrimaryKeys("TargetId")
	sla.SetNonUniqueKeys("InventoryType")
	sla.SetArgs(p)

	vnic.Resources().Registry().Register(&l8tpollaris.TargetAction{})

	ws := web.New(ServiceName, ServiceArea, 0)
	ws.AddEndpoint(&l8tpollaris.L8PTarget{}, ifs.POST, &l8web.L8Empty{})
	ws.AddEndpoint(&l8tpollaris.L8PTargetList{}, ifs.POST, &l8web.L8Empty{})
	ws.AddEndpoint(&l8tpollaris.TargetAction{}, ifs.POST, &l8web.L8Empty{})
	ws.AddEndpoint(&l8tpollaris.L8PTarget{}, ifs.PUT, &l8web.L8Empty{})
	ws.AddEndpoint(&l8tpollaris.L8PTarget{}, ifs.PATCH, &l8web.L8Empty{})
	ws.AddEndpoint(&l8api.L8Query{}, ifs.DELETE, &l8web.L8Empty{})
	ws.AddEndpoint(&l8api.L8Query{}, ifs.GET, &l8tpollaris.L8PTargetList{})
	sla.SetWebService(ws)

	vnic.Resources().Services().Activate(sla, vnic)

	go callback.InitTargets(vnic)
}

// Targets retrieves the Targets service handler from the service registry.
// Returns the handler and true if found, nil and false otherwise.
func Targets(vnic ifs.IVNic) (ifs.IServiceHandler, bool) {
	return vnic.Resources().Services().ServiceHandler(ServiceName, ServiceArea)
}

// Target retrieves a single target by its ID from the database.
// Returns the target and nil error on success, or nil and an error
// if the service is not found or the target doesn't exist.
func Target(targetId string, vnic ifs.IVNic) (*l8tpollaris.L8PTarget, error) {
	this, ok := Targets(vnic)
	if !ok {
		return nil, errors.New("No Targets Service Found")
	}
	filter := &l8tpollaris.L8PTarget{TargetId: targetId}
	resp := this.Get(object.New(nil, filter), vnic)
	if resp.Error() != nil {
		return nil, resp.Error()
	}
	return resp.Element().(*l8tpollaris.L8PTarget), nil
}

// openDBConection establishes a connection to the PostgreSQL database.
// It uses localhost (127.0.0.1) on port 5432 with SSL disabled.
// Panics if the connection cannot be established or ping fails.
func openDBConection(dbname, user, pass, port string) *sql.DB {
	p, _ := strconv.Atoi(port)
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		"127.0.0.1", p, user, pass, dbname)
	db, err := sql.Open("postgres", psqlInfo)

	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(fmt.Errorf("failed to connect to database: %w", err))
	}

	return db
}
