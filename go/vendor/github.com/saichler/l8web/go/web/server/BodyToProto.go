/*
 * Copyright (c) 2025 Sharon Aicler (saichler@gmail.com)
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

// BodyToProto.go provides utilities for parsing HTTP request bodies into
// Protocol Buffer messages. It handles JSON unmarshaling via protojson
// and returns appropriate HTTP error responses for parsing failures.

package server

import (
	"fmt"
	"io"
	"net/http"
	"reflect"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// bodyToProto reads an HTTP request body and unmarshals it into a Protocol Buffer message.
// It handles JSON input via protojson and returns appropriate HTTP error responses for failures.
//
// Parameters:
//   - w: HTTP response writer for error responses
//   - r: HTTP request to read body from
//   - method: HTTP method name for error messages
//   - body: Target Protocol Buffer message to unmarshal into
//
// Returns true if parsing succeeded, false if an error occurred (error already written to response).
// Empty request bodies are allowed and will leave the body message in its zero state.
func bodyToProto(w http.ResponseWriter, r *http.Request, method string, body proto.Message) bool {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to read body for method " + method + "\n"))
		w.Write([]byte(err.Error()))
		fmt.Println("Failed to read body for method " + method + "\n")
		return false
	}

	if data != nil && len(data) > 0 {
		err = protojson.Unmarshal(data, body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Failed to unmarshal body for method " + method + " element " + reflect.ValueOf(body).Elem().Type().Name() + "\n"))
			w.Write([]byte("body for method " + method + string(data) + "\n"))
			w.Write([]byte(err.Error()))
			fmt.Println("Failed to unmarshal body for method " + method + " element " + reflect.ValueOf(body).Elem().Type().Name() + "\n")
			fmt.Println("body for method " + method + string(data) + "\n")
			return false
		}
	}
	return true
}
