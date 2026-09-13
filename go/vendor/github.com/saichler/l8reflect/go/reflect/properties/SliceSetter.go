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

// This file contains slice-specific setter logic for property operations.
// Handles slice creation, resizing, element insertion, and deletion through property paths.

package properties

import (
	"errors"
	"reflect"

	"github.com/saichler/l8types/go/ifs"
)

const maxTimeSeriesPoints = 100

// timeSeriesAppend appends time series points to the slice.
// Handles both a single *L8TimeSeriesPoint and a full []*L8TimeSeriesPoint slice.
// If the slice reaches maxTimeSeriesPoints, the oldest entries are dropped.
func (this *Property) timeSeriesAppend(myValue reflect.Value, newPoint reflect.Value) (interface{}, error) {
	info, err := this.resources.Registry().Info(this.node.TypeName)
	if err != nil {
		return nil, err
	}
	// If the slice doesn't exist yet, create it
	if !myValue.IsValid() || myValue.IsNil() {
		myValue.Set(reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(info.Type())), 0, 0))
	}

	// If the incoming value is a slice, append each element individually
	if newPoint.Kind() == reflect.Slice {
		for i := 0; i < newPoint.Len(); i++ {
			if myValue.Len() >= maxTimeSeriesPoints {
				myValue.Index(0).Set(reflect.Zero(myValue.Type().Elem()))
				myValue.Set(myValue.Slice(1, myValue.Len()))
			}
			myValue.Set(reflect.Append(myValue, newPoint.Index(i)))
		}
		return myValue.Interface(), nil
	}

	// Single point: drop oldest if at capacity, then append
	if myValue.Len() >= maxTimeSeriesPoints {
		myValue.Index(0).Set(reflect.Zero(myValue.Type().Elem()))
		myValue.Set(myValue.Slice(1, myValue.Len()))
	}

	// Append the new point
	myValue.Set(reflect.Append(myValue, newPoint))
	return myValue.Interface(), nil
}

// sliceSet handles setting values within slice fields.
// Supports replacing entire slices, setting individual elements by index,
// creating new slices, resizing for larger indices, and handling deletions.
func (this *Property) sliceSet(myValue reflect.Value, newSliceValue reflect.Value) (interface{}, error) {
	//Replace all the slice
	if this.key == nil {
		// Handle setting slice to nil or a new slice
		if newSliceValue.Kind() == reflect.Slice || !newSliceValue.IsValid() {
			// Check if myValue is valid and settable
			if myValue.IsValid() && myValue.CanSet() {
				if !newSliceValue.IsValid() {
					// Setting to nil - create a zero value of the appropriate slice type
					sliceType := myValue.Type()
					nilSlice := reflect.Zero(sliceType)
					myValue.Set(nilSlice)
					return nil, nil
				} else {
					myValue.Set(newSliceValue)
					return myValue.Interface(), nil
				}
			} else {
				// If we can't set the value, just return the new value
				if newSliceValue.IsValid() {
					return newSliceValue.Interface(), nil
				}
				return nil, nil
			}
		}
	}

	// Check if this.key is nil before casting
	if this.key == nil {
		return nil, nil // Return nil for setting nil on slice without index
	}

	index := this.key.(int)
	info, err := this.resources.Registry().Info(this.node.TypeName)
	if err != nil {
		return nil, err
	}

	//If this is a new slice
	if !myValue.IsValid() || myValue.IsNil() {
		if this.node.IsStruct {
			myValue.Set(reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(info.Type())), index+1, index+1))
		} else {
			myValue.Set(reflect.MakeSlice(reflect.SliceOf(info.Type()), index+1, index+1))
		}
	}

	//If elements were delete from the slice,
	//reduce the size of the slice
	if newSliceValue.Kind() == reflect.String && newSliceValue.String() == ifs.Deleted_Entry {
		var newSlice reflect.Value
		if this.node.IsStruct {
			newSlice = reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(info.Type())), index, index)
		} else {
			newSlice = reflect.MakeSlice(reflect.SliceOf(info.Type()), index, index)
		}
		for i := 0; i < index; i++ {
			newSlice.Index(i).Set(myValue.Index(i))
		}
		myValue.Set(newSlice)
		return myValue.Interface(), nil
	}

	//If the index is larger than the current slice, enlarge it
	if index >= myValue.Len() {
		var newSlice reflect.Value
		if this.node.IsStruct {
			newSlice = reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(info.Type())), index+1, index+1)
		} else {
			newSlice = reflect.MakeSlice(reflect.SliceOf(info.Type()), index+1, index+1)
		}
		for i := 0; i < myValue.Len(); i++ {
			newSlice.Index(i).Set(myValue.Index(i))
		}
		myValue.Set(newSlice)
	}

	oIndexValue := myValue.Index(index)

	if this.node.IsStruct && (!oIndexValue.IsValid() || oIndexValue.IsNil()) {
		oIndexValue.Set(reflect.New(info.Type()))
	}

	if this.node.IsStruct && !this.IsLeaf() {
		return oIndexValue.Interface(), nil
	}

	if newSliceValue.Kind() != reflect.Slice {
		pid, _ := this.PropertyId()
		return nil, errors.New("No a slice new value PID: " + pid)
	}

	nIndexValue := newSliceValue.Index(index)

	//If this is not a leaf property
	//We need to continue drilling down
	if this.node.IsStruct && !this.IsLeaf() {
		if !oIndexValue.IsValid() {
			o, _ := info.NewInstance()
			oIndexValue.Set(reflect.ValueOf(o))
		}
		return oIndexValue.Interface(), nil
	}

	oIndexValue.Set(nIndexValue)

	return oIndexValue.Interface(), err
}
