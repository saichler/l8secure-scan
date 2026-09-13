# Object Serialization Library

A high-performance Go serialization library that provides deep object serialization with Protocol Buffers integration and dynamic type registry support.

## Features

- **Type-Safe Serialization**: Automatic type detection and handling using Go's reflection system
- **Protocol Buffers Integration**: Native support for protobuf message serialization
- **Dynamic Type Registry**: Runtime type resolution for complex object graphs
- **Query Support**: Built-in GSQL query language integration for selective serialization
- **Binary Format**: Efficient binary encoding with automatic buffer management
- **Base64 Encoding**: Built-in text encoding support for network transmission
- **Null Safety**: Proper handling of nil values and empty collections

## Architecture

### Core Components

The library consists of several key components:

- **Object**: Main serialization engine with buffer management
- **Elements**: Container for multiple serialized objects with query support
- **Type Handlers**: Specialized serializers for primitive and complex types
- **Registry**: Dynamic type resolution system for deserialization

### Supported Types

#### Primitive Types
- Integers: `int`, `int32`, `int64`, `uint32`, `uint64`
- Floating Point: `float32`, `float64`
- Text: `string`
- Boolean: `bool`

#### Complex Types
- **Structs**: Serialized using Protocol Buffers
- **Slices**: Dynamic arrays with type preservation
- **Maps**: Key-value collections with type inference
- **Pointers**: Automatic dereferencing with null handling

## Usage

### Basic Serialization

```go
import "github.com/saichler/l8srlz/go/serialize/object"

// Serialize an object
func serialize(data interface{}) ([]byte, error) {
    return object.DataOf(data)
}

// Deserialize an object
func deserialize(data []byte, registry ifs.IRegistry) (interface{}, error) {
    return object.ElemOf(data, registry)
}
```

### Working with Elements

```go
// Create elements container
elements := object.New(nil, myData)

// Serialize multiple elements
data, err := elements.Serialize()
if err != nil {
    return err
}

// Deserialize elements
err = elements.Deserialize(data, registry)
if err != nil {
    return err
}
```

### Query Integration

```go
// Create query-based elements
elements, err := object.NewQuery("SELECT * FROM users WHERE age > 18", resources)
if err != nil {
    return err
}
```

### Base64 Encoding

```go
// Encode to Base64
obj := object.NewEncode()
obj.Add(myData)
base64String := obj.Base64()

// Decode from Base64
data, err := object.FromBase64(base64String)
if err != nil {
    return err
}
```

## Protocol Buffers Integration

The library seamlessly integrates with Protocol Buffers for struct serialization:

```go
// Your protobuf message
message := &pb.MyMessage{
    Field1: "value1",
    Field2: 42,
}

// Serialize (automatically uses protobuf)
data, err := object.DataOf(message)

// Deserialize (requires type registry)
result, err := object.ElemOf(data, registry)
```

## Type Registry

For deserialization of complex types, register your types with the registry:

```go
// Register protobuf types
registry.Register("MyMessage", &pb.MyMessage{})

// Now deserialization will work
result, err := object.ElemOf(data, registry)
```

## Error Handling

The library provides comprehensive error handling:

```go
// Create elements with error
elements := object.NewError("Custom error message")

// Check for errors
if elements.Error() != nil {
    log.Printf("Error: %v", elements.Error())
}
```

## Performance Considerations

- **Buffer Management**: Automatic buffer expansion minimizes allocations
- **Type Caching**: Reflection results are cached for improved performance
- **Binary Format**: Compact binary representation reduces payload size
- **Zero-Copy**: Efficient byte slice operations where possible

## Thread Safety

The library is designed for concurrent use with proper synchronization. Individual `Object` instances are not thread-safe and should not be shared across goroutines without external synchronization.

## Dependencies

- `google.golang.org/protobuf/proto` - Protocol Buffers support
- `github.com/saichler/l8types/go/ifs` - Interface definitions
- `github.com/saichler/l8types/go/types` - Type system support
- `github.com/saichler/gsql/go/gsql/interpreter` - Query language support

## License

© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at: http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

## Contributing

Contributions are welcome! Please see the main project README for contributing guidelines.