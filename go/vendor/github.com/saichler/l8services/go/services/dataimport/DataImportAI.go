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

package dataimport

import (
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8reflect"
	"reflect"
	"sort"
)

func (h *AIHandler) Post(elems ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	req, ok := elems.Element().(*l8api.L8ImportAIMappingRequest)
	if !ok {
		return object.NewError("invalid AI mapping request type")
	}

	node, found := vnic.Resources().Introspector().NodeByTypeName(req.TargetModelType)
	if !found {
		return object.NewError("unknown model type: " + req.TargetModelType)
	}

	fields := buildFieldInfoList(node, vnic.Resources().Introspector())

	resp, err := GetAIMappingProvider().SuggestMapping(fields, req.SourceColumns, req.SampleValues)
	if err != nil {
		return object.NewError("AI mapping failed: " + err.Error())
	}

	return object.New(nil, resp)
}

// buildFieldInfoList extracts importable field metadata from a node's attributes.
func buildFieldInfoList(node *l8reflect.L8Node, introspector ifs.IIntrospector) []*l8api.L8ImportFieldInfo {
	var fields []*l8api.L8ImportFieldInfo

	for name, attr := range node.Attributes {
		if attr.IsStruct || attr.IsSlice || attr.IsMap {
			continue
		}

		info := &l8api.L8ImportFieldInfo{
			FieldName: name,
			FieldType: kindToFieldType(introspector.Kind(attr)),
		}

		isPK := introspector.Decorators().BoolDecoratorValueForNode(
			attr, l8reflect.L8DecoratorType_Primary)
		info.IsPrimaryKey = isPK

		fields = append(fields, info)
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].FieldName < fields[j].FieldName
	})

	return fields
}

// kindToFieldType maps a reflect.Kind to a human-readable type string.
func kindToFieldType(k reflect.Kind) string {
	switch k {
	case reflect.String:
		return "string"
	case reflect.Int32:
		return "int32"
	case reflect.Int64:
		return "int64"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.Bool:
		return "bool"
	case reflect.Uint32, reflect.Uint64:
		return "uint"
	default:
		return "string"
	}
}
