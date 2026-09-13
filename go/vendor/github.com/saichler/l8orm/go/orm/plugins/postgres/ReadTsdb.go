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
package postgres

import (
	"reflect"
	"strings"

	"github.com/saichler/l8orm/go/orm/common"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8reflect"
)

func (this *Postgres) populateTsFields(result ifs.IElements, resources ifs.IResources) ifs.IElements {
	if result == nil || result.Error() != nil {
		return result
	}
	elements := result.Elements()
	if len(elements) == 0 {
		return result
	}

	sample := elements[0]
	if sample == nil {
		return result
	}
	typeName := reflect.TypeOf(sample).Elem().Name()
	node, ok := resources.Introspector().Node(typeName)
	if !ok {
		return result
	}

	tsAttrs := collectTsAttrs(node)
	if len(tsAttrs) == 0 {
		return result
	}

	for _, elem := range elements {
		if elem == nil {
			continue
		}
		v := reflect.ValueOf(elem)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if !v.IsValid() {
			continue
		}

		key, _, _ := resources.Introspector().Decorators().PrimaryKeyDecoratorFromValue(node, v)
		if key == "" {
			continue
		}
		prefix := strings.ToLower(typeName) + "<" + key + ">"

		for attrName := range tsAttrs {
			propertyId := prefix + "." + strings.ToLower(attrName)
			points, err := this.tsdb.GetTSDBLatest(propertyId, 100)
			if err != nil || len(points) == 0 {
				continue
			}
			field := v.FieldByName(attrName)
			if !field.IsValid() || !field.CanSet() {
				continue
			}
			slice := reflect.MakeSlice(field.Type(), len(points), len(points))
			for i, p := range points {
				slice.Index(i).Set(reflect.ValueOf(p))
			}
			field.Set(slice)
		}
	}
	return result
}

func collectTsAttrs(node *l8reflect.L8Node) map[string]*l8reflect.L8Node {
	tsAttrs := make(map[string]*l8reflect.L8Node)
	for attrName, attrNode := range node.Attributes {
		if attrNode.IsStruct && common.IsTimeSeriesType(attrNode.TypeName) {
			tsAttrs[attrName] = attrNode
		}
	}
	return tsAttrs
}
