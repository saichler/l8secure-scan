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
package object

import (
	"errors"
	"reflect"

	"github.com/saichler/l8ql/go/gsql/interpreter"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
)

// Elements is a container for multiple serializable objects with support for
// queries, metadata, notifications, and replica tracking. It implements the
// ifs.IElements interface and is commonly used for transmitting collections
// of data between microservices.
//
// Elements supports:
//   - Multiple element storage with key-value pairs
//   - Error tracking per element
//   - L8QL query integration for selective data retrieval
//   - Metadata for statistics and pagination
//   - Notification flags for event-driven systems
//   - Replica support for distributed systems
type Elements struct {
	elements     []*Element        // Slice of element wrappers
	query        ifs.IQuery        // Parsed query object
	pquery       *l8api.L8Query    // Protocol buffer query representation
	metadata     *l8api.L8MetaData // Metadata including counts and pagination info
	notification bool              // Flag indicating this is a notification

	isReplica bool // Flag indicating this is a replica request
	replica   byte // Replica number for distributed systems
}

// Element wraps a single value with its associated key and any error
// that occurred during processing. This allows batch operations to
// track success/failure per item.
type Element struct {
	element interface{} // The actual data value
	key     interface{} // Optional key for map-like access
	error   error       // Error associated with this element, if any
}

// NewQuery creates a new Elements container with an L8QL query string.
// The query is parsed and stored for later execution. If the query string
// is empty, returns an empty Elements container with an empty query.
//
// Parameters:
//   - gsql: L8QL query string (e.g., "SELECT * FROM users WHERE age > 18")
//   - resources: Resources containing registry and other dependencies
//
// Returns the Elements container and any parsing error.
func NewQuery(gsql string, resources ifs.IResources) (ifs.IElements, error) {
	if gsql == "" {
		return New(nil, &l8api.L8Query{}), nil
	}
	q, e := interpreter.NewQuery(gsql, resources)
	if e != nil {
		return nil, e
	}
	elems := &Elements{pquery: q.Query()}
	return elems, nil
}

// NewNotify creates a new Elements container marked as a notification.
// Notifications are used in event-driven systems to signal that data
// has changed and should be processed accordingly.
func NewNotify(any interface{}) ifs.IElements {
	elems := New(nil, any)
	elems.(*Elements).notification = true
	return elems
}

// NewReplicaRequest creates a replica request from existing elements.
// This is used in distributed systems to send data to replica nodes
// with a specific replica number for identification.
//
// Parameters:
//   - elems: The source elements to replicate
//   - replica: The replica number (0-255) identifying the target replica
func NewReplicaRequest(elems ifs.IElements, replica byte) ifs.IElements {
	c := clone(elems)
	c.replica = replica
	c.isReplica = true
	return c
}

// clone creates a shallow copy of an Elements container.
// The element slice is shared, not copied.
func clone(e ifs.IElements) *Elements {
	old := e.(*Elements)
	c := &Elements{}
	c.elements = old.elements
	c.query = old.query
	c.pquery = old.pquery
	c.notification = old.notification
	c.replica = old.replica
	return c
}

// New creates a new Elements container from any Go value.
// It automatically handles slices and maps by extracting their elements
// into the internal element list. For other types, the value is stored
// as a single element.
//
// Parameters:
//   - err: Optional error to associate with the first element
//   - any: The data to store (can be a single value, slice, or map)
//
// Panics if 'any' is a function, as this is likely a programming error.
func New(err error, any interface{}) ifs.IElements {

	if reflect.ValueOf(any).Kind() == reflect.Func {
		panic("any is a function, this is probably a mistake")
	}

	result := &Elements{}
	result.elements = make([]*Element, 1)
	result.elements[0] = &Element{}
	if err != nil {
		result.elements[0].error = err
	}

	v := reflect.ValueOf(any)
	if v.IsValid() {
		if v.Kind() == reflect.Slice {
			result.elements = nil
			for i := 0; i < v.Len(); i++ {
				result.Add(v.Index(i).Interface(), i, nil)
			}
		} else if v.Kind() == reflect.Map {
			result.elements = nil
			keys := v.MapKeys()
			for _, key := range keys {
				result.Add(v.MapIndex(key).Interface(), key.Interface(), nil)
			}
		} else {
			result.elements[0].element = any
		}
	}

	return result
}

// NewQueryResult creates a new Elements container with pre-existing metadata.
// This is typically used when returning query results that include statistics
// like total count, pagination info, etc.
func NewQueryResult(any interface{}, metadata *l8api.L8MetaData) ifs.IElements {
	elements := New(nil, any)
	elements.(*Elements).metadata = metadata
	return elements
}

// NewError creates a new Elements container containing only an error.
// This is a convenience function for returning error responses.
func NewError(err string) ifs.IElements {
	return New(errors.New(err), nil)
}

// Query returns the parsed query object, initializing it from the protocol
// buffer representation if necessary. This lazy initialization allows the
// query to be transmitted efficiently and parsed only when needed.
func (this *Elements) Query(resources ifs.IResources) (ifs.IQuery, error) {
	var err error
	if this.query == nil && this.pquery != nil {
		this.query, err = interpreter.NewFromQuery(this.pquery, resources)
		if err != nil {
			return nil, err
		}
	}
	return this.query, nil
}

// Add appends a new element to the container with an optional key and error.
// This allows building up collections incrementally.
func (this *Elements) Add(elem interface{}, key interface{}, err error) {
	mobject := &Element{element: elem, key: key, error: err}
	if this.elements == nil {
		this.elements = make([]*Element, 0)
	}
	this.elements = append(this.elements, mobject)
}

// Elements returns all stored values as a slice of interface{}.
func (this *Elements) Elements() []interface{} {
	result := make([]interface{}, len(this.elements))
	for i, o := range this.elements {
		result[i] = o.element
	}
	return result
}

// Element returns the first stored value, or nil if the container is empty.
// This is convenient when the container holds a single item.
func (this *Elements) Element() interface{} {
	if this.elements == nil || len(this.elements) == 0 {
		return nil
	}
	return this.elements[0].element
}

// Keys returns all stored keys as a slice of interface{}.
func (this *Elements) Keys() []interface{} {
	result := make([]interface{}, len(this.elements))
	for i, o := range this.elements {
		result[i] = o.key
	}
	return result
}

// Key returns the key of the first element.
func (this *Elements) Key() interface{} {
	return this.elements[0].key
}

// Errors returns all errors associated with elements as a slice.
func (this *Elements) Errors() []error {
	result := make([]error, len(this.elements))
	for i, o := range this.elements {
		result[i] = o.error
	}
	return result
}

// Error returns the error of the first element, or nil if empty.
// This is convenient for checking if a single-element container has an error.
func (this *Elements) Error() error {
	if this.elements == nil || len(this.elements) == 0 {
		return nil
	}
	return this.elements[0].error
}

// Serialize converts the entire Elements container to a byte slice.
// The format includes:
//   - Element count
//   - For each element: value, key, error message (empty string if no error)
//   - Metadata (auto-generated if not set)
//   - Query (if present)
//
// Returns the serialized bytes and any error encountered.
func (this *Elements) Serialize() ([]byte, error) {
	obj := NewEncode()
	obj.Add(len(this.elements))
	var err error

	for _, o := range this.elements {
		err = obj.Add(o.element)
		if err != nil {
			return nil, err
		}
		err = obj.Add(o.key)
		if err != nil {
			return nil, err
		}
		if o.error != nil {
			err = obj.Add(o.error.Error())
		} else {
			err = obj.Add("")
		}
		if err != nil {
			return nil, err
		}
	}

	if this.metadata == nil {
		this.metadata = &l8api.L8MetaData{}
		this.metadata.KeyCount = &l8api.L8Count{}
		this.metadata.KeyCount.Counts = make(map[string]float64)
		this.metadata.KeyCount.Counts["Total"] = float64(len(this.elements))
	}
	obj.Add(this.metadata)

	obj.Add(this.pquery)
	return obj.Data(), nil
}

// PQuery returns the protocol buffer query representation.
func (this *Elements) PQuery() *l8api.L8Query {
	return this.pquery
}

// Deserialize reconstructs the Elements container from a byte slice.
// It reverses the Serialize() operation, restoring all elements,
// metadata, and query information.
//
// Parameters:
//   - data: The serialized byte slice
//   - r: Type registry for resolving complex types
func (this *Elements) Deserialize(data []byte, r ifs.IRegistry) error {
	location := 0
	obj := NewDecode(data, location, r)

	s, err := obj.Get()
	if err != nil {
		return err
	}
	size := s.(int)
	this.elements = make([]*Element, size)
	var eMsg interface{}
	for i := 0; i < size; i++ {
		elem := &Element{}
		elem.element, err = obj.Get()
		if err != nil {
			return err
		}
		elem.key, err = obj.Get()
		if err != nil {
			return err
		}
		eMsg, err = obj.Get()
		if err != nil {
			return err
		}
		errMsg := eMsg.(string)
		if errMsg != "" {
			elem.error = errors.New(errMsg)
		}
		this.elements[i] = elem
	}

	st, err := obj.Get()
	if err != nil {
		return err
	}
	this.metadata, _ = st.(*l8api.L8MetaData)

	pq, err := obj.Get()
	if err != nil {
		return err
	}
	this.pquery, _ = pq.(*l8api.L8Query)
	return nil
}

// Notification returns true if this Elements container is marked as a notification.
func (this *Elements) Notification() bool {
	return this.notification
}

// Replica returns the replica number (0-255) for distributed system operations.
func (this *Elements) Replica() byte {
	return this.replica
}

// IsReplica returns true if this Elements container is a replica request.
func (this *Elements) IsReplica() bool {
	return this.isReplica
}

// Append adds all elements from another Elements container to this one.
// Keys and errors are not preserved; only the values are copied.
func (this *Elements) Append(elements ifs.IElements) {
	if elements == nil {
		return
	}
	if elements.Elements() == nil {
		return
	}
	for _, elem := range elements.Elements() {
		this.Add(elem, nil, nil)
	}
}

// AsList converts the elements to a typed list structure registered in the
// registry. It looks for a type named "<ElementType>List" and populates its
// "List" field with the elements.
//
// This is useful for Protocol Buffers message types that follow the convention
// of having a list wrapper type.
//
// Returns the first element if no list type is registered, or an error if
// the container is empty.
func (this *Elements) AsList(r ifs.IRegistry) (interface{}, error) {
	if len(this.elements) == 0 {
		return nil, errors.New("elements is empty")
	}
	if this.elements[0] == nil || this.elements[0].element == nil {
		return nil, errors.New("element is nil")
	}
	listName := reflect.ValueOf(this.elements[0].element).Elem().Type().Name() + "List"
	info, err := r.Info(listName)

	if err != nil {
		return this.elements[0].element, nil
	}

	listItem, err := info.NewInstance()
	if err != nil {
		return nil, err
	}
	v := reflect.ValueOf(listItem).Elem()
	f := v.FieldByName("List")
	newList := reflect.MakeSlice(f.Type(), len(this.elements), len(this.elements))
	for i := 0; i < len(this.elements); i++ {
		newList.Index(i).Set(reflect.ValueOf(this.elements[i].element))
	}
	f.Set(newList)

	f = v.FieldByName("Metadata")
	if f.IsValid() && f.CanSet() {
		f.Set(reflect.ValueOf(this.metadata))
	}

	return listItem, nil
}

// Metadata returns the metadata associated with this elements collection.
func (this *Elements) Metadata() *l8api.L8MetaData {
	return this.metadata
}

// IsFilterMode returns true if the container is operating in filter mode,
// which means it has no query but contains elements that can be used as
// filter criteria.
func (this *Elements) IsFilterMode() bool {
	if this.pquery == nil && (this.elements != nil || len(this.elements) == 1) {
		return true
	}
	return false
}
