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
package convert

import (
	"errors"
	"github.com/saichler/l8orm/go/orm/common"
	"github.com/saichler/l8orm/go/types/l8orms"
	"strings"
	"sync"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8reflect"
)

// NewRelationsDataForQuery creates an L8OrmRData structure initialized for a query.
// It builds the table schema based on the query's root type and requested properties.
// Only the tables needed for the query's projections are included.
func NewRelationsDataForQuery(query ifs.IQuery) (*l8orms.L8OrmRData, error) {
	rlData := &l8orms.L8OrmRData{}
	rlData.RootTypeName = query.RootType().TypeName
	rlData.Tables = make(map[string]*l8orms.L8OrmTable)
	properties := make([]string, len(query.Properties()))
	for i, p := range query.Properties() {
		id, _ := p.PropertyId()
		index := strings.Index(id, ".")
		properties[i] = id[index+1:]
	}
	err := addTable(query.RootType(), rlData, properties...)
	return rlData, err
}

// NewRelationalDataForType creates an L8OrmRData structure for a given type name.
// It builds the complete table schema including all nested struct types.
// This is useful for creating the relational structure without a query context.
func NewRelationalDataForType(typeName string, introspector ifs.IIntrospector) (*l8orms.L8OrmRData, error) {
	node, ok := introspector.NodeByTypeName(typeName)
	if !ok {
		return nil, errors.New("Did not find any node for " + typeName)
	}
	rlData := &l8orms.L8OrmRData{}
	rlData.RootTypeName = typeName
	rlData.Tables = make(map[string]*l8orms.L8OrmTable)
	err := addTable(node, rlData, typeName)
	return rlData, err
}

// addTable adds a table to the relational data structure for the given node.
// If properties are specified, only those columns are added; otherwise all non-struct
// attributes become columns. Nested struct attributes trigger recursive table creation.
func addTable(node *l8reflect.L8Node, rlData *l8orms.L8OrmRData, properties ...string) error {
	_, ok := rlData.Tables[node.TypeName]
	if ok && properties == nil {
		return nil
	}
	table := &l8orms.L8OrmTable{}
	table.Name = node.TypeName
	rlData.Tables[node.TypeName] = table

	if properties == nil || len(properties) == 0 {
		SetColumns(table, node)
		for _, attr := range node.Attributes {
			if attr.IsStruct {
				if common.IsTimeSeriesType(attr.TypeName) {
					continue
				}
				err := addTable(attr, rlData)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	subProperties := make([]string, 0)
	for _, property := range properties {
		attrName, subProperty := getAttrName(property, node)
		if subProperty != "" {
			subProperties = append(subProperties, property)
			continue
		}
		attr := node.Attributes[attrName]
		if attr.IsStruct {
			return errors.New("Trying to get attribute " + attr.FieldName + " from " +
				node.TypeName + ", which is a table")
		}
		addColumn(table, attr.FieldName)
	}
	for _, subProperty := range subProperties {
		attrName, property := getAttrName(subProperty, node)
		attr := node.Attributes[attrName]
		err := addTable(attr, rlData, property)
		if err != nil {
			return err
		}
	}
	return nil
}

// attributes caches lowercase-to-original attribute name mappings per type.
// This enables case-insensitive property name resolution.
var attributes = make(map[string]map[string]string)

// mtx protects the attributes cache for concurrent access.
var mtx = &sync.RWMutex{}

// getAttrName resolves a property name to its actual attribute name in the node.
// It handles dotted property paths (e.g., "Sub.Field") by extracting the first component
// and returning the remainder as the sub-property. Case-insensitive matching is used.
func getAttrName(property string, node *l8reflect.L8Node) (string, string) {
	var attrName string
	var attrProp string
	index := strings.Index(property, ".")
	if index != -1 {
		attrName = property[0:index]
		attrProp = property[index+1:]
	} else {
		attrName = property
	}

	if node.Attributes[attrName] != nil {
		return attrName, attrProp
	}

	attrName = strings.ToLower(attrName)
	mtx.RLock()
	defer mtx.RUnlock()
	_, ok := attributes[node.TypeName]
	if !ok {
		mtx.RUnlock()
		mtx.Lock()
		attributes[node.TypeName] = make(map[string]string)
		for name, _ := range node.Attributes {
			attributes[node.TypeName][strings.ToLower(name)] = name
		}
		mtx.Unlock()
		mtx.RLock()
	}
	attrName = attributes[node.TypeName][attrName]
	if attrName == "" {
		panic(property + " does not exist")
	}
	return attrName, attrProp
}

// SetColumns initializes the column map for a table based on the node's attributes.
// Each non-struct attribute becomes a column with a unique index.
// This function is idempotent - it only sets columns if they haven't been set.
func SetColumns(table *l8orms.L8OrmTable, node *l8reflect.L8Node) {
	if table.Columns == nil {
		table.Columns = make(map[string]int32)
		for attrName, attrNode := range node.Attributes {
			if attrNode.IsStruct {
				continue
			}
			table.Columns[attrName] = int32(len(table.Columns) + 1)
		}
	}
}

// addColumn adds a single column to a table's schema.
// The column index is automatically assigned based on the current column count.
func addColumn(table *l8orms.L8OrmTable, attrName string) {
	if table.Columns == nil {
		table.Columns = make(map[string]int32)
	}
	table.Columns[attrName] = int32(len(table.Columns) + 1)
}
