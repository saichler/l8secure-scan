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

// MessageUnmarshal.go provides message deserialization from bytes.
// Uses unsafe string conversion for zero-copy performance.

package ifs

import (
	"unsafe"
)

// unsafeString converts bytes to string without copying (zero-copy).
// Warning: The returned string shares memory with the byte slice.
func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Unmarshal deserializes a message from bytes received over the network.
// Decrypts the body using the security provider and populates all fields.
// Note: Uses string() instead of unsafeString to copy data and allow GC of original buffers.
func (this *Message) Unmarshal(data []byte, resources IResources) (interface{}, error) {

	this.source = string(data[pSource:pVnet])
	this.vnet = string(data[pVnet:pDestination])
	this.destination = toDestinationSafe(data)
	this.serviceName = toServiceNameSafe(data)
	this.serviceArea = data[pServiceArea]
	this.priority, this.multicastMode = ByteToPriorityMulticastMode(data[PPriority])

	body, err := resources.Security().Decrypt(string(data[PPriority+1:]))
	if err != nil {
		return nil, err
	}

	this.action = Action(body[pAction])
	this.tr_state = TransactionState(body[pTrState])
	this.aaaId = string(body[pAaaId:pSequence])
	this.sequence = Bytes2UInt32(body[pSequence:pTimeout])
	this.timeout = Bytes2UInt16(body[pTimeout:pRequestReply])
	this.request, this.reply = BoolOf(body[pRequestReply])

	failMessageSize := int(body[pFailMessageSize])
	pDataSize := pFailMessage + failMessageSize
	this.failMessage = string(body[pFailMessage:pDataSize])

	pData := pDataSize + sUint32
	dataSize := int(Bytes2UInt32(body[pDataSize:pData]))
	pTrId := pData + dataSize
	// Copy data slice to allow GC of the decrypted body buffer
	this.data = append([]byte(nil), body[pData:pTrId]...)

	if this.tr_state != NotATransaction {
		pTrErrMsgSize := pTrId + sUuid
		this.tr_id = string(body[pTrId:pTrErrMsgSize])
		trErrMsgSize := int(body[pTrErrMsgSize])
		pTrErrMsg := pTrErrMsgSize + sByte
		pTrCreated := pTrErrMsg + trErrMsgSize
		pTrQueued := pTrCreated + 8
		pTrRunning := pTrQueued + 8
		pTrEnd := pTrRunning + 8
		pTrTimeout := pTrEnd + 8
		pTrReplica := pTrTimeout + 8
		pTrIsReplica := pTrReplica + sByte
		this.tr_errMsg = string(body[pTrErrMsg:pTrCreated])
		this.tr_created = Bytes2Long(body[pTrCreated:pTrQueued])
		this.tr_queued = Bytes2Long(body[pTrQueued:pTrRunning])
		this.tr_running = Bytes2Long(body[pTrRunning:pTrEnd])
		this.tr_end = Bytes2Long(body[pTrEnd:pTrTimeout])
		this.tr_timeout = Bytes2Long(body[pTrTimeout:pTrReplica])
		this.tr_replica = body[pTrReplica]
		this.tr_isReplica = body[pTrIsReplica] == 1
	}

	return nil, nil
}

// HeaderOf extracts header fields from raw message bytes without full deserialization.
// Returns: source, vnet, destination, serviceName, serviceArea, priority, multicastMode
func HeaderOf(data []byte) (string, string, string, string, byte, Priority, MulticastMode) {
	return unsafeString(data[pSource:pVnet]),
		unsafeString(data[pVnet:pDestination]),
		ToDestination(data),
		ToServiceName(data),
		data[pServiceArea],
		Priority(data[PPriority] >> 4),
		MulticastMode(data[PPriority] & 0x0F)
}

// ToDestination extracts the destination UUID from raw message bytes.
// Returns empty string if destination is not set (multicast messages).
func ToDestination(data []byte) string {
	if data[pDestination] != 0 && data[pDestination+1] != 0 {
		return unsafeString(data[pDestination:pServiceName])
	}
	return ""
}

// ToServiceName extracts the service name from raw message bytes.
// Handles null-terminated strings within the fixed-size field.
func ToServiceName(data []byte) string {
	start := pServiceName
	end := start + sServiceName
	if end > len(data) {
		end = len(data)
	}

	for i := start; i < end; i++ {
		if data[i] == 0 {
			return unsafeString(data[start:i])
		}
	}
	return unsafeString(data[start:end])
}

// toDestinationSafe extracts destination with a copy (for Unmarshal where data is retained).
func toDestinationSafe(data []byte) string {
	if data[pDestination] != 0 && data[pDestination+1] != 0 {
		return string(data[pDestination:pServiceName])
	}
	return ""
}

// toServiceNameSafe extracts service name with a copy (for Unmarshal where data is retained).
func toServiceNameSafe(data []byte) string {
	start := pServiceName
	end := start + sServiceName
	if end > len(data) {
		end = len(data)
	}

	for i := start; i < end; i++ {
		if data[i] == 0 {
			return string(data[start:i])
		}
	}
	return string(data[start:end])
}
