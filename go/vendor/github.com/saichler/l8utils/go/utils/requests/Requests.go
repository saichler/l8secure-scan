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

// Package requests provides request/response coordination for asynchronous operations.
// It tracks pending requests and matches incoming responses using message source and number.
//
// Key features:
//   - Thread-safe request tracking using sync.Map
//   - Configurable timeout per request with automatic cleanup
//   - Duplicate request detection and rejection
//   - Context cancellation support
//   - Transaction-aware response handling
package requests

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8services"
	"github.com/saichler/l8utils/go/utils/strings"
)

var (
	ErrDuplicateRequest = errors.New("duplicate request")
)

// Requests manages pending request/response pairs using a lock-free sync.Map.
type Requests struct {
	pending sync.Map // map[string]*Request - eliminates mutex bottleneck
}

// Request represents a pending request awaiting a response with timeout support.
type Request struct {
	msgSource      string
	msgNum         uint32
	timeout        time.Duration
	timeoutReached bool
	response       ifs.IElements
	log            ifs.ILogger
	responseChan   chan ifs.IElements
	cancel         context.CancelFunc
	once           sync.Once
	mtx            sync.RWMutex
	requests       *Requests // reference to parent for auto-cleanup
}

// NewRequests creates a new request tracker.
func NewRequests() *Requests {
	return &Requests{}
}

// NewRequest creates and registers a new pending request. Returns ErrDuplicateRequest if
// a request with the same msgNum and msgSource already exists.
func (this *Requests) NewRequest(msgNum uint32, msgSource string, timeoutInSeconds int, log ifs.ILogger) (*Request, error) {
	key := requestKey(msgSource, msgNum)

	request := &Request{
		msgNum:       msgNum,
		msgSource:    msgSource,
		timeout:      time.Duration(timeoutInSeconds) * time.Second,
		log:          log,
		responseChan: make(chan ifs.IElements, 1),
		requests:     this,
	}

	// Check for duplicate and store atomically
	if _, loaded := this.pending.LoadOrStore(key, request); loaded {
		return nil, ErrDuplicateRequest
	}

	return request, nil
}

// GetRequest retrieves a pending request by its message number and source.
func (this *Requests) GetRequest(msgNum uint32, msgSource string) *Request {
	key := requestKey(msgSource, msgNum)
	val, ok := this.pending.Load(key)
	if !ok {
		return nil
	}
	return val.(*Request)
}

// DelRequest removes a request from tracking.
func (this *Requests) DelRequest(msgNum uint32, msgSource string) {
	key := requestKey(msgSource, msgNum)
	this.pending.Delete(key)
}

// MsgNum returns the message number for this request.
func (this *Request) MsgNum() uint32 {
	return this.msgNum
}

// MsgSource returns the message source identifier.
func (this *Request) MsgSource() string {
	return this.msgSource
}

// Response returns the received response or a timeout error if applicable.
func (this *Request) Response() ifs.IElements {
	this.mtx.RLock()
	defer this.mtx.RUnlock()

	if this.timeoutReached {
		return object.NewError("Timeout Reached!")
	}
	return this.response
}

// Wait blocks until a response is received or the timeout expires.
// Automatically cleans up the request from tracking when done.
func (this *Request) Wait() ifs.IElements {
	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	this.cancel = cancel
	defer cancel()
	defer this.cleanup()

	select {
	case resp := <-this.responseChan:
		this.mtx.Lock()
		this.response = resp
		this.mtx.Unlock()
		return resp
	case <-ctx.Done():
		this.mtx.Lock()
		this.timeoutReached = true
		this.mtx.Unlock()
		return object.NewError("Timeout Reached!")
	}
}

// SetResponse delivers a response to the waiting caller. For transactions,
// only the final response (with End != 0) completes the request.
func (this *Request) SetResponse(resp ifs.IElements) {
	if this == nil || resp == nil {
		return
	}

	tr, ok := resp.Element().(*l8services.L8Transaction)

	// If transaction is not ended, don't complete the request yet
	if ok && tr.End == 0 {
		return
	}

	// Use once to ensure we only send the final response once
	this.once.Do(func() {
		// Send response to channel (non-blocking)
		select {
		case this.responseChan <- resp:
			// Response sent successfully
		default:
			// Channel already has a response or is closed
		}
	})
}

func (this *Request) cleanup() {
	// Auto-cleanup: remove from pending map
	if this.requests != nil {
		this.requests.DelRequest(this.msgNum, this.msgSource)
	}

	// Close the response channel to free resources
	close(this.responseChan)

	// Cancel context if not already canceled
	if this.cancel != nil {
		this.cancel()
	}
}

func requestKey(msgSource string, msgNum uint32) string {
	return strings.New(msgSource, int(msgNum)).String()
}
