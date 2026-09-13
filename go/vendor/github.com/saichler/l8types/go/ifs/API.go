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

// Package ifs (interfaces) defines the core interfaces for the Layer 8 distributed system.
// It provides abstractions for data elements, queries, messaging, services, security,
// and network communication. These interfaces form the foundation for building
// distributed applications with the Layer 8 ecosystem.
package ifs

import (
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8reflect"
)

const (
	// Deleted_Entry is a sentinel value marking deleted entries in the distributed cache.
	Deleted_Entry = "__DD__"
)

// IElements represents a collection of data elements with associated keys and errors.
// It provides methods for serialization, querying, and handling replicated data.
// This is the primary interface for working with data in the Layer 8 system.
type IElements interface {
	// Elements returns all data elements in this collection.
	Elements() []interface{}
	// Keys returns the keys corresponding to each element.
	Keys() []interface{}
	// Errors returns any errors associated with this collection.
	Errors() []error
	// Element returns a single element (first in collection).
	Element() interface{}
	// Query creates a query object from this elements collection.
	Query(IResources) (IQuery, error)
	// Key returns the key of the first element.
	Key() interface{}
	// Error returns the first error if any.
	Error() error
	// Serialize converts this collection to bytes.
	Serialize() ([]byte, error)
	// Deserialize populates this collection from bytes.
	Deserialize([]byte, IRegistry) error
	// Notification returns true if this is a notification payload.
	Notification() bool
	// Append adds elements from another IElements collection.
	Append(IElements)
	// AsList converts elements to a typed list.
	AsList(IRegistry) (interface{}, error)
	// IsFilterMode returns true if this is in filter/query mode.
	IsFilterMode() bool
	// IsReplica returns true if this is replicated data.
	IsReplica() bool
	// Replica returns the replica number (0 for primary).
	Replica() byte
	// Metadata returns the metadata associated with this collection.
	Metadata() *l8api.L8MetaData
}

// IQuery represents a query for filtering and retrieving data elements.
// Supports criteria-based filtering, sorting, pagination, and map-reduce operations.
type IQuery interface {
	// RootType returns the root type node for this query.
	RootType() *l8reflect.L8Node
	// Properties returns the list of properties to include in results.
	Properties() []IProperty
	// Criteria returns the filter expression tree.
	Criteria() IExpression
	// KeyOf returns the key property name.
	KeyOf() string
	// Match tests if an object matches this query's criteria.
	Match(interface{}) bool
	// Page returns the current page number for pagination.
	Page() int32
	// Limit returns the maximum results per page.
	Limit() int32
	// SortBy returns the property name to sort by.
	SortBy() string
	// SortByValue extracts the sort value from an object.
	SortByValue(interface{}) interface{}
	// MatchCase returns true if matching is case-sensitive.
	MatchCase() bool
	// Descending returns true if sorting in descending order.
	Descending() bool
	// MapReduce returns true if this is a map-reduce query.
	MapReduce() bool
	// Text returns the free-text search string.
	Text() string
	// Hash returns a unique hash for this query (for caching).
	Hash() int32
	// ValueForParameter retrieves a parameter value by name.
	ValueForParameter(string) string
	// Aggregates returns the parsed aggregate functions (COUNT, SUM, AVG, MIN, MAX) from the SELECT clause.
	Aggregates() []*l8api.L8AggregateFunction
	// GroupBy returns the list of fields to group results by.
	GroupBy() []string
	// Having returns the filter expression applied to aggregated results.
	Having() IExpression
	// IsAggregate returns true if this query has aggregate functions.
	IsAggregate() bool
	// Filter returns the subset of items matching the WHERE clause.
	Filter([]interface{}, bool) []interface{}
	// Aggregate groups items by GROUP BY fields and computes aggregate functions.
	// Returns one map per group with aggregate alias keys and group-by field values.
	Aggregate([]interface{}) []map[string]interface{}
	// Register returns true if the client opts into real-time change notifications.
	Register() bool
	// AAAId returns the authenticated user's identity for this query.
	AAAId() string
	// SetAAAId sets the authenticated user's identity on this query.
	SetAAAId(string)
}

// IProperty represents a property accessor for a type field.
// Provides methods to get/set values and inspect the property metadata.
type IProperty interface {
	// PropertyId returns the unique identifier for this property.
	PropertyId() (string, error)
	// Get retrieves the property value from an object.
	Get(interface{}) (interface{}, error)
	// Set sets the property value on an object, returning old and new values.
	Set(interface{}, interface{}) (interface{}, interface{}, error)
	// Node returns the reflection node for this property.
	Node() *l8reflect.L8Node
	// Parent returns the parent property (for nested properties).
	Parent() IProperty
	// IsString returns true if this property is a string type.
	IsString() bool
	// Resources returns the resources context.
	Resources() IResources
}

// IExpression represents a node in a query filter expression tree.
// Expressions can be chained with logical operators and nested.
type IExpression interface {
	// Condition returns the condition at this expression node.
	Condition() ICondition
	// Operator returns the logical operator (AND/OR) to the next expression.
	Operator() string
	// Next returns the next expression in the chain.
	Next() IExpression
	// Child returns the nested child expression.
	Child() IExpression
}

// ICondition represents a condition within an expression.
// Contains comparators that can be chained together.
type ICondition interface {
	// Comparator returns the comparison operation.
	Comparator() IComparator
	// Operator returns the operator to chain with next condition.
	Operator() string
	// Next returns the next condition in the chain.
	Next() ICondition
}

// IComparator defines a single comparison operation.
// Compares a left operand against a right operand using an operator.
type IComparator interface {
	// Left returns the left operand string (typically a property name).
	Left() string
	// LeftProperty returns the left operand as a property accessor.
	LeftProperty() IProperty
	// Right returns the right operand string (the value to compare).
	Right() string
	// RightProperty returns the right operand as a property accessor.
	RightProperty() IProperty
	// Operator returns the comparison operator (=, !=, >, <, etc.).
	Operator() string
}
