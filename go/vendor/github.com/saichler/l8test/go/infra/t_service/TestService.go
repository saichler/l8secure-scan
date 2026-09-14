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
	"errors"
	"sync"
	"sync/atomic"

	"github.com/saichler/l8services/go/services/dcache"
	. "github.com/saichler/l8srlz/go/serialize/object"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/testtypes"
	"github.com/saichler/l8utils/go/utils/web"
)

type TestServiceBase struct {
	name         string
	postNumber   atomic.Int32
	putNumber    atomic.Int32
	patchNumber  atomic.Int32
	deleteNumber atomic.Int32
	getNumber    atomic.Int32
	failedNumber atomic.Int32
	errorMode    bool
	getReplica   *sync.Map
	postReplica  *sync.Map
}

const (
	ServiceName    = "Tests"
	ServiceType    = "TestServiceHandler"
	ServiceTrType  = "TestServiceTransactionHandler"
	ServiceRepType = "TestServiceReplicationHandler"
)

var test2Handler *TestServiceReplicationHandler

func (this *TestServiceHandler) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	this.name = sla.Args()[0].(string)
	return nil
}

func (this *TestServiceTransactionHandler) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	this.name = sla.Args()[0].(string)
	return nil
}

func (this *TestServiceReplicationHandler) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	this.name = sla.Args()[0].(string)
	vnic.Resources().Introspector().Decorators().AddPrimaryKeyDecorator(&testtypes.TestProto{}, "MyString")
	this.cache = dcache.NewReplicationCache(vnic.Resources(), nil)
	this.postReplica = &sync.Map{}
	this.getReplica = &sync.Map{}
	test2Handler = this
	return nil
}
func (this *TestServiceBase) DeActivate() error {
	return nil
}

func (this *TestServiceBase) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	Log.Debug("Post -", this.name, "- Test callback")
	if pb.IsReplica() {
		key := test2Handler.TransactionConfig().KeyOf(pb, vnic.Resources())
		v, ok := this.postReplica.Load(key)
		if !ok {
			this.postReplica.Store(key, 1)
		} else {
			this.postReplica.Store(key, v.(int)+1)
		}
		test2Handler.cache.Post(pb.Element(), int(pb.Replica()))
	} else {
		this.postNumber.Add(1)
	}
	var err error
	if this.errorMode {
		err = errors.New("Post - TestServiceBase Error")
	}
	return New(err, pb.Element())
}
func (this *TestServiceBase) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	Log.Debug("Put -", this.name, "- Test callback")
	this.putNumber.Add(1)
	var err error
	if this.errorMode {
		err = errors.New("Put - TestServiceBase Error")
	}
	return New(err, pb.Element())
}
func (this *TestServiceBase) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	Log.Debug("Patch -", this.name, "- Test callback")
	this.patchNumber.Add(1)
	var err error
	if this.errorMode {
		err = errors.New("Patch - TestServiceBase Error")
	}
	return New(err, pb.Element())
}
func (this *TestServiceBase) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	Log.Debug("Delete -", this.name, "- Test callback")
	this.deleteNumber.Add(1)
	var err error
	if this.errorMode {
		err = errors.New("Delete - TestServiceBase Error")
	}
	return New(err, pb.Element())
}

func (this *TestServiceBase) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	Log.Debug("Get -", this.name, "- Test callback")
	if pb.IsReplica() {
		key := test2Handler.TransactionConfig().KeyOf(pb, vnic.Resources())
		v, ok := this.getReplica.Load(key)
		if !ok {
			this.getReplica.Store(key, 1)
		} else {
			this.getReplica.Store(key, v.(int)+1)
		}
		elem, _ := test2Handler.cache.Get(pb.Element(), int(pb.Replica()))
		return New(nil, elem)
	} else {
		this.getNumber.Add(1)
	}
	var err error
	if this.errorMode {
		err = errors.New("Get - TestServiceBase Error")
	}
	return New(err, pb.Element())
}
func (this *TestServiceBase) Failed(pb ifs.IElements, vnic ifs.IVNic, info *ifs.Message) ifs.IElements {
	dest := "n/a"
	msg := "n/a"
	if info != nil {
		dest = info.Source()
		msg = info.FailMessage()
	}
	Log.Debug("Failed -", this.name, " to ", dest, "- Test callback")
	Log.Debug("Failed Reason is ", msg)
	this.failedNumber.Add(1)
	var err error
	if this.errorMode {
		err = errors.New("Failed - TestServiceBase Error")
	}
	return New(err, pb.Element())
}
func (this *TestServiceBase) End() string {
	return "/Tests"
}
func (this *TestServiceBase) ServiceName() string {
	return ServiceName
}
func (this *TestServiceBase) ServiceModel() ifs.IElements {
	return New(nil, &testtypes.TestProto{})
}

type TestServiceHandler struct {
	TestServiceBase
}

func (this *TestServiceHandler) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

func (this *TestServiceHandler) WebService() ifs.IWebService {
	pb := &testtypes.TestProto{}
	pblist := &testtypes.TestProtoList{}
	webSvr := web.New(ServiceName, 0, 0)
	webSvr.AddEndpoint(pb, ifs.POST, pb)
	webSvr.AddEndpoint(pb, ifs.PUT, pb)
	webSvr.AddEndpoint(pb, ifs.PATCH, pb)
	webSvr.AddEndpoint(pb, ifs.DELETE, pb)
	webSvr.AddEndpoint(pb, ifs.GET, pblist)
	return webSvr
}

func (this *TestServiceHandler) Merge(mapr map[string]ifs.IElements) ifs.IElements {
	results := make([]interface{}, 0)
	for uuid, elems := range mapr {
		if elems.Error() != nil {
			Log.Error("MapReduce Merge: participant ", uuid, " returned error: ", elems.Error().Error())
			continue
		}
		for _, elem := range elems.Elements() {
			if elem == nil {
				Log.Error("MapReduce Merge: participant ", uuid, " returned nil element")
				continue
			}
			results = append(results, elem)
		}
	}
	if len(results) != len(mapr) {
		Log.Error("MapReduce Merge: expected ", len(mapr), " results, got ", len(results),
			" (", len(mapr)-len(results), " participants failed)")
	}
	return New(nil, results)
}

type TestServiceTransactionHandler struct {
	TestServiceBase
}

func (this *TestServiceTransactionHandler) TransactionConfig() ifs.ITransactionConfig {
	return this
}

func (this *TestServiceTransactionHandler) Replication() bool {
	return false
}
func (this *TestServiceTransactionHandler) ReplicationCount() int {
	return 0
}
func (this *TestServiceTransactionHandler) KeyOf(elements ifs.IElements, resources ifs.IResources) string {
	return ""
}
func (this *TestServiceTransactionHandler) Voter() bool {
	return true
}
func (this *TestServiceTransactionHandler) WebService() ifs.IWebService {
	pb := &testtypes.TestProto{}
	webSvr := web.New(ServiceName, 0, 0)
	webSvr.AddEndpoint(pb, ifs.POST, pb)
	webSvr.AddEndpoint(pb, ifs.PUT, pb)
	webSvr.AddEndpoint(pb, ifs.PATCH, pb)
	webSvr.AddEndpoint(pb, ifs.DELETE, pb)
	webSvr.AddEndpoint(pb, ifs.GET, pb)
	return webSvr
}

type TestServiceReplicationHandler struct {
	TestServiceBase
	cache ifs.IReplicationCache
}

func (this *TestServiceReplicationHandler) TransactionConfig() ifs.ITransactionConfig {
	return this
}

func (this *TestServiceReplicationHandler) Replication() bool {
	return true
}
func (this *TestServiceReplicationHandler) ReplicationCount() int {
	return 2
}
func (this *TestServiceReplicationHandler) Voter() bool {
	return true
}
func (this *TestServiceReplicationHandler) KeyOf(elements ifs.IElements, resources ifs.IResources) string {
	pb := elements.Element().(*testtypes.TestProto)
	return pb.MyString
}
func (this *TestServiceReplicationHandler) WebService() ifs.IWebService {
	pb := &testtypes.TestProto{}
	pblist := &testtypes.TestProtoList{}
	webSvr := web.New(ServiceName, 0, 0)
	webSvr.AddEndpoint(pb, ifs.POST, pb)
	webSvr.AddEndpoint(pb, ifs.PUT, pb)
	webSvr.AddEndpoint(pb, ifs.PATCH, pb)
	webSvr.AddEndpoint(pb, ifs.DELETE, pb)
	webSvr.AddEndpoint(pb, ifs.GET, pblist)
	return webSvr
}
