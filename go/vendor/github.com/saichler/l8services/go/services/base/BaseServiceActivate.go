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

package base

import (
	"errors"
	"github.com/saichler/l8services/go/services/recovery"
	"github.com/saichler/l8utils/go/utils/queues"
	"reflect"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/cache"
)

// Activate is the public entry point for activating a BaseService. It registers
// the BaseService type, activates the service through the service manager, and
// initiates a recovery check in a background goroutine to ensure data consistency.
func Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) (ifs.IServiceHandler, error) {
	vnic.Resources().Registry().Register(&BaseService{})
	//return vnic.Resources().Services().Activate(sla, vnic)
	b, e := vnic.Resources().Services().Activate(sla, vnic)
	bs := b.(*BaseService)
	go recovery.RecoveryCheck(sla.ServiceName(), sla.ServiceArea(), bs.cache.ModelType(), vnic)
	return b, e
}

// Activate initializes the BaseService with the provided SLA and virtual NIC.
// For stateful services, it sets up the primary key decorator, creates the cache
// with persistence store, registers metadata functions, and starts the notification
// processing goroutine. Panics if service is stateless without a callback.
func (this *BaseService) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	this.sla = sla
	this.running = true
	if !sla.Stateful() && sla.Callback() == nil {
		panic("Nothing to do when stateless and no callback")
	}
	if this.sla.Stateful() {
		err := vnic.Resources().Introspector().Decorators().AddPrimaryKeyDecorator(sla.ServiceItem(), sla.PrimaryKeys()...)
		if err != nil {
			return err
		}
		this.cache = cache.NewCache(this.sla.ServiceItem(), this.sla.InitItems(),
			this.sla.Store(), vnic.Resources())
		if sla.MetadataFunc() != nil {
			for name, f := range sla.MetadataFunc() {
				this.cache.AddMetadataFunc(name, f)
			}
		}
		if !this.sla.Transactional() {
			this.nQueue = queues.NewQueue(sla.ServiceName(), 10000)
			this.cache.SetNotificationsFor(sla.ServiceName(), sla.ServiceArea())
			this.vnic = vnic
			go this.processNotificationQueue()
		}
	}
	return nil
}

// DeActivate gracefully shuts down the service by calling Shutdown.
// This stops the notification queue processing and releases resources.
func (this *BaseService) DeActivate() error {
	this.Shutdown()
	return nil
}

// validateElem checks that the provided element is valid and matches the
// expected model type for this service's cache. Returns an error if the
// element is invalid or has an incorrect type.
func (this *BaseService) validateElem(pb ifs.IElements) error {
	v := reflect.ValueOf(pb.Element())
	if !v.IsValid() {
		return errors.New("Invalid element ")
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Type().Name() != this.cache.ModelType() {
		return errors.New("Invalid element type " + v.Type().Name())
	}
	return nil
}
