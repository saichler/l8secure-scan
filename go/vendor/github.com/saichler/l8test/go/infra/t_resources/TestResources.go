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

package t_resources

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/saichler/l8bus/go/overlay/protocol"
	"github.com/saichler/l8reflect/go/reflect/cloning"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/testtypes"
	"github.com/saichler/l8utils/go/utils/logger"
	"github.com/saichler/l8utils/go/utils/shared"
)

const (
	VNET_PREFIX = "vnet-"
	VNIC_PREFIX = "-vnic-"
)

var Log = logger.NewLoggerDirectImpl(&logger.FmtLogMethod{})

func CreateResources(vnetPort, vnicNum int, level ifs.LogLevel) (ifs.IResources, string) {
	alias := AliasOf(vnetPort, vnicNum)
	res := shared.ResourcesOf(alias, uint32(vnetPort), 0)
	res.Logger().SetLogLevel(level)
	res.Introspector().Decorators().AddPrimaryKeyDecorator(&testtypes.TestProto{}, "MyString")
	res.Registry().Register(&testtypes.TestProtoList{})
	return res, alias
}

func AliasOf(vnetPort, vnicNum int) string {
	alias := bytes.Buffer{}
	alias.WriteString(VNET_PREFIX)
	alias.WriteString(strconv.Itoa(vnetPort))
	if vnicNum != -1 {
		alias.WriteString(VNIC_PREFIX)
		alias.WriteString(strconv.Itoa(vnicNum))
	}
	return alias.String()
}

func CreateSubSubModelInstance(index1, index2, index3, k int, v ...int32) *testtypes.TestProtoSubSub {
	key := bytes.Buffer{}
	key.WriteString("string-")
	key.WriteString(strconv.Itoa(index1))
	key.WriteString("-")
	key.WriteString(strconv.Itoa(index2))
	key.WriteString("-")
	key.WriteString(strconv.Itoa(index3))
	subsub := &testtypes.TestProtoSubSub{}

	subsub.MyString = key.String()
	subsub.MyInt64 = time.Now().UnixNano()
	subsub.Int32Map = make(map[int32]int32)
	for i := 0; i < k; i++ {
		subsub.Int32Map[int32(i)] = v[i]
	}
	return subsub
}

func CreateSubTestModelInstance(index1, index2 int, n, k int, v ...int32) *testtypes.TestProtoSub {
	key := bytes.Buffer{}
	key.WriteString("string-")
	key.WriteString(strconv.Itoa(index1))
	key.WriteString("-")
	key.WriteString(strconv.Itoa(index2))
	sub := &testtypes.TestProtoSub{}
	sub.MyString = key.String()
	sub.MyInt64 = time.Now().UnixNano()
	sub.MySubs = make(map[string]*testtypes.TestProtoSubSub)

	for i := 0; i < n; i++ {
		subsub := CreateSubSubModelInstance(index1, index2, i, k, v...)
		sub.MySubs[subsub.MyString] = subsub
	}

	return sub
}

func CreateTestModelInstance(index int) *testtypes.TestProto {
	key := bytes.Buffer{}
	key.WriteString("string-")
	key.WriteString(strconv.Itoa(index))

	sub1m := CreateSubTestModelInstance(index, 1, 2, 2, 0, 0)
	sub2m := CreateSubTestModelInstance(index, 2, 2, 2, 0, 1)
	sub3m := CreateSubTestModelInstance(index, 3, 2, 4, 0, 1, 2, 3)

	i := &testtypes.TestProto{
		MyString:           key.String(),
		MyFloat64:          123456.123456,
		MyBool:             true,
		MyFloat32:          123.123,
		MyInt32:            int32(index),
		MyInt64:            int64(index * 10),
		MyInt32Slice:       []int32{1, 2, 3, int32(index)},
		MyStringSlice:      []string{"a", "b", "c", "d", key.String()},
		MyInt32ToInt64Map:  map[int32]int64{1: 11, 2: 22, 3: 33, 4: 44, int32(index): int64(index * 10)},
		MyString2StringMap: map[string]string{"a": "aa", "b": "bb", "c": "cc", key.String(): key.String() + key.String()},
		MySingle:           CreateSubTestModelInstance(index, 0, 2, 2, 0, 0),
		MyModelSlice: []*testtypes.TestProtoSub{CreateSubTestModelInstance(index, 1, 2, 2, 0, 0),
			CreateSubTestModelInstance(index, 2, 2, 2, 0, 1)},
		MyString2ModelMap: map[string]*testtypes.TestProtoSub{sub1m.MyString: sub1m,
			sub2m.MyString: sub2m, sub3m.MyString: sub3m},
		MyEnum: testtypes.TestEnum_ValueOne,
	}
	return i
}

func CloneTestModel(a *testtypes.TestProto) *testtypes.TestProto {
	cloner := cloning.NewCloner()
	return cloner.Clone(a).(*testtypes.TestProto)
}

func WaitForCondition(cond func() bool, timeoutInSeconds int64, t *testing.T, failMessage string) bool {
	fmt.Println("Messages Created Start:", protocol.MsgLog.Total())
	defer func() {
		fmt.Println("Messages Created End:", protocol.MsgLog.Total())
	}()
	start := time.Now().UnixMilli()
	end := start + timeoutInSeconds*1000
	for start < end {
		if cond() {
			return true
		}
		time.Sleep(time.Millisecond * 100)
		start += 100
	}
	if t != nil {
		Log.Fail(t, failMessage)
	} else {
		Log.Error(failMessage)
	}
	return false
}
