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
package persist

import (
	"errors"
	"reflect"

	"github.com/saichler/l8ql/go/gsql/interpreter"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8reflect"
	"github.com/saichler/l8utils/go/utils/strings"
)

// ElementToQuery converts a filter element into an IQuery for database operations.
// It extracts the primary key values from the element and constructs a query
// that selects all records matching those key values. This enables "query by example"
// where an object with some fields set is used as a filter template.
func ElementToQuery(pb ifs.IElements, elem interface{}, vnic ifs.IVNic) (ifs.IQuery, error) {
	asideValue := reflect.ValueOf(pb.Element())
	aside := asideValue.Elem().Type().Name()
	bside := reflect.ValueOf(elem).Elem().Type().Name()
	if aside == bside {
		rnode, ok := vnic.Resources().Introspector().NodeByTypeName(bside)
		if ok {
			gsql := strings.New("select * from ", bside, " where ")
			fields, e := vnic.Resources().Introspector().Decorators().Fields(rnode, l8reflect.L8DecoratorType_Primary)
			if e != nil {
				return nil, e
			}
			for i, field := range fields {
				if i > 0 {
					gsql.Add(" and ")
				}
				v := asideValue.Elem().FieldByName(field)
				gsql.Add(field)
				gsql.Add("=")
				if v.Kind() == reflect.String {
					gsql.Add("'", v.String(), "'")
				} else {
					gsql.Add(v.String())
				}
			}
			q, e := interpreter.NewQuery(gsql.String(), vnic.Resources())
			return q, e
		}
	}
	return nil, errors.New("Element does not match " + bside + " != " + aside)
}

// KeyOf extracts the primary key value from elements using the introspector's decorators.
// This utility function provides a consistent way to get the identifying key for an element
// regardless of how the primary key is defined on the type.
func KeyOf(elements ifs.IElements, resources ifs.IResources) string {
	key, _, err := resources.Introspector().Decorators().PrimaryKeyDecoratorValue(elements.Element())
	if err != nil {
		resources.Logger().Error(err.Error())
	}
	return key
}
