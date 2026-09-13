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

// Package base provides the foundational service implementation for the Layer 8 services framework.
// It implements core CRUD operations (Post, Put, Patch, Delete, Get) with integrated caching,
// transaction support, and notification handling.
package base

import (
	"github.com/saichler/l8reflect/go/reflect/updating"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8web"
	"github.com/saichler/l8utils/go/utils/cache"
	"github.com/saichler/l8utils/go/utils/queues"
)

// BaseService provides the core service handler implementation that manages
// CRUD operations with caching, SLA enforcement, and notification support.
// It serves as the foundation for distributed services in the Layer 8 ecosystem.
type BaseService struct {
	cache   *cache.Cache
	vnic    ifs.IVNic
	sla     *ifs.ServiceLevelAgreement
	nQueue  *queues.Queue
	running bool
}

// Post creates new elements in the service cache. It delegates to the do method
// with the POST action, which handles callback invocations and notification generation.
func (this *BaseService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return this.do(ifs.POST, pb, vnic)
}

// Put performs a full replacement update of elements in the service cache.
// It delegates to the do method with the PUT action.
func (this *BaseService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return this.do(ifs.PUT, pb, vnic)
}

// Patch performs a partial update of elements in the service cache.
// Only the specified fields are modified while preserving other fields.
func (this *BaseService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return this.do(ifs.PATCH, pb, vnic)
}

// Delete removes elements from the service cache.
// It delegates to the do method with the DELETE action.
func (this *BaseService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return this.do(ifs.DELETE, pb, vnic)
}

// Get retrieves elements from the service cache. It supports two modes:
// - Filter mode: retrieves a single element matching the provided filter
// - Query mode: retrieves multiple elements with pagination support
// Filter mode invokes the SLA callback's Before/After hooks with the single
// unwrapped domain element being filtered on, the same per-element contract
// "do" uses for POST/PUT/PATCH/DELETE. Query mode has no discrete domain
// element (only a parsed query), so — mirroring "do"'s own behavior on an
// empty element set — the callback is not invoked.
func (this *BaseService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	if this.cache == nil {
		return pb
	}
	if pb.IsFilterMode() {
		return this.getFiltered(pb, vnic)
	}
	return this.getQueried(pb, vnic)
}

// getFiltered handles single-element filter GETs (e.g. get-by-id). The SLA
// callback's Before/After hooks receive the unwrapped filter element, not
// the surrounding IElements container.
func (this *BaseService) getFiltered(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	filterElem := pb.Element()

	if this.sla.Callback() != nil && filterElem != nil {
		beforElem, cont, err := this.sla.Callback().Before(filterElem, ifs.GET, false, vnic)
		if err != nil {
			return object.NewError(err.Error())
		}
		if !cont {
			return object.New(nil, &l8web.L8Empty{})
		}
		if beforElem != nil {
			filterElem = beforElem
		}
	}

	e := this.validateElem(pb)
	if e != nil {
		return object.New(e, &l8web.L8Empty{})
	}
	resp, err := this.cache.Get(filterElem)
	if this.sla.Callback() != nil {
		if vnic != nil {
			upd := updating.NewUpdater(vnic.Resources(), false, false)
			upd.Update(resp, filterElem)
		}
		after, _, _ := this.sla.Callback().After(resp, ifs.GET, true, vnic)
		if after != nil {
			resp = after
		}
	}
	return object.New(err, resp)
}

// getQueried handles bulk L8Query GETs (e.g. "select * from X"). There is no
// discrete domain element to hand to the SLA callback, so — matching "do"'s
// behavior when its element loop is empty — the callback is not invoked here.
func (this *BaseService) getQueried(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	resources := vnic.Resources()
	if this.vnic != nil {
		resources = this.vnic.Resources()
	}
	q, e := pb.Query(resources)
	if e != nil {
		return object.NewError(e.Error())
	}
	elems, counts := this.cache.Fetch(int(q.Page()*q.Limit()), int(q.Limit()), q)
	return object.NewQueryResult(elems, counts)
}

// Failed handles message delivery failures by logging an error.
// It is invoked when a message cannot be delivered to its intended recipient.
func (this *BaseService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	resources := vnic.Resources()
	if this.vnic != nil {
		resources = this.vnic.Resources()
	}
	resources.Logger().Error("Failed to deliver message")
	return nil
}

// TransactionConfig returns the transaction configuration for this service.
// Returns nil if the service is stateless or non-transactional; otherwise
// returns this service instance as the transaction configuration.
func (this *BaseService) TransactionConfig() ifs.ITransactionConfig {
	if !this.sla.Stateful() {
		return nil
	}
	if this.sla.Transactional() {
		return this
	}
	return nil
}

// WebService returns the web service interface from the SLA configuration,
// used for exposing the service via HTTP/REST endpoints.
func (this *BaseService) WebService() ifs.IWebService {
	return this.sla.WebService()
}

// Replication returns whether data replication is enabled for this service
// as specified in the SLA configuration.
func (this *BaseService) Replication() bool {
	return this.sla.Replication()
}

// ReplicationCount returns the number of replicas configured for this service
// as specified in the SLA configuration.
func (this *BaseService) ReplicationCount() int {
	return this.sla.ReplicationCount()
}

// KeyOf extracts and returns the primary key value from the given elements
// using the introspector's primary key decorator.
func (this *BaseService) KeyOf(elems ifs.IElements, r ifs.IResources) string {
	key, _, _ := r.Introspector().Decorators().PrimaryKeyDecoratorValue(elems.Element())
	return key
}

// Voter returns whether this service participates in leader election voting.
func (this *BaseService) Voter() bool {
	return this.sla.Voter()
}
