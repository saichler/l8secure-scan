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

package registry

import (
	"errors"
	"github.com/saichler/l8types/go/ifs"
	"reflect"
)

/*
Type info contains only two attributes, the golang type of, usually, a struct.
And a Serializer, which indicates a which serializer should be used to transalte
this type into bytes. Serializer can be nil...
*/
type Info struct {
	/* The golang reflect type */
	typ reflect.Type
	/* The serializers */
	serializers map[ifs.SerializerMode]ifs.ISerializer
}

// NewInfo /* Constructs a new type info with the given attributes */
func NewInfo(typ reflect.Type) (*Info, error) {
	if typ == nil {
		return nil, errors.New("Cannot register a nil type")
	}
	return &Info{typ: typ,
		serializers: make(map[ifs.SerializerMode]ifs.ISerializer)}, nil
}

// Type /* Return the reflect type of this TypeInfo */
func (info *Info) Type() reflect.Type {
	return info.typ
}

// Name /* Return the name of the type/struct */
func (info *Info) Name() string {
	return info.typ.Name()
}

// Serializer /* Return the serializer to be used with this type. Can be nil... */
func (info *Info) Serializer(mode ifs.SerializerMode) ifs.ISerializer {
	return info.serializers[mode]
}

func (info *Info) AddSerializer(serializer ifs.ISerializer) {
	info.serializers[serializer.Mode()] = serializer
}

// NewInstance /* Construct a new instance via reflect using the type */
func (info *Info) NewInstance() (interface{}, error) {
	n := reflect.New(info.typ)
	if !n.IsValid() {
		return nil, errors.New("was not able to create new instance of type " + info.typ.Name())
	}
	return n.Interface(), nil
}
