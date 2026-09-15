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

package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8notify"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type wsConn struct {
	conn *websocket.Conn
	wmu  sync.Mutex
}

func (c *wsConn) writeJSON(data []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// WebSocketManager manages WebSocket connections keyed by AAAId (authenticated user identity).
type WebSocketManager struct {
	mu          sync.RWMutex
	connections map[string]*wsConn
	vnic        ifs.IVNic
}

func NewWebSocketManager(vnic ifs.IVNic) *WebSocketManager {
	return &WebSocketManager{
		connections: make(map[string]*wsConn),
		vnic:        vnic,
	}
}

// HandleUpgrade validates the bearer token, resolves the AAAId, and upgrades to a WebSocket connection.
func (this *WebSocketManager) HandleUpgrade(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	aaaId, ok := this.vnic.Resources().Security().ValidateToken(token, this.vnic)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	wc := &wsConn{conn: conn}

	this.mu.Lock()
	old, exists := this.connections[aaaId]
	if exists {
		old.conn.Close()
	}
	this.connections[aaaId] = wc
	this.mu.Unlock()

	go this.writePump(wc)
	go this.readPump(aaaId, wc)
}

// readPump reads from the connection to detect close and handle pings.
func (this *WebSocketManager) readPump(aaaId string, wc *wsConn) {
	defer this.Remove(aaaId)
	wc.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	wc.conn.SetPongHandler(func(string) error {
		wc.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, _, err := wc.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// writePump sends periodic pings to keep the connection alive.
func (this *WebSocketManager) writePump(wc *wsConn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		wc.wmu.Lock()
		err := wc.conn.WriteMessage(websocket.PingMessage, nil)
		wc.wmu.Unlock()
		if err != nil {
			return
		}
	}
}

// Remove closes and removes the connection for the given AAAId.
func (this *WebSocketManager) Remove(aaaId string) {
	this.mu.Lock()
	wc, ok := this.connections[aaaId]
	if ok {
		wc.conn.Close()
		delete(this.connections, aaaId)
	}
	this.mu.Unlock()
}

// OnNotification serializes a notification and sends to subscribed clients.
func (this *WebSocketManager) OnNotification(notification *l8notify.L8NotificationSet) {
	action := ""
	switch notification.Type {
	case l8notify.L8NotificationType_Post:
		action = "add"
	case l8notify.L8NotificationType_Put, l8notify.L8NotificationType_Patch:
		action = "update"
	case l8notify.L8NotificationType_Delete:
		action = "delete"
	default:
		return
	}

	msg := map[string]interface{}{
		"action":     action,
		"modelType":  notification.ModelType,
		"primaryKey": notification.ModelKey,
	}

	// Forward the actual record too, when it can be reconstructed, so the
	// client's existing msg.record-aware handling (Layer8DTable's update
	// branch) can patch the row in place instead of always re-fetching
	// (l8utils/plans/generic-websocket-change-notifications.md Phase 4).
	// This is the client notification (cn), not the cross-node delta
	// notification (n) -- Cache.notifications.go always puts the *whole*
	// item into NotificationList[0] regardless of action (even for Patch,
	// which deliberately reuses the whole-item Add encoder, not a
	// per-property encoding), so decode directly rather than reusing
	// notify.ItemOf (which assumes per-property NotificationList entries
	// for Patch, matching the *other*, differently-built notification).
	if len(notification.NotificationList) > 0 {
		entry := notification.NotificationList[0]
		raw := entry.NewValue
		if raw == nil {
			raw = entry.OldValue // delete
		}
		if raw != nil {
			obj := object.NewDecode(raw, 0, this.vnic.Resources().Registry())
			if item, derr := obj.Get(); derr == nil && item != nil {
				if pbItem, ok := item.(proto.Message); ok {
					marshalOptions := protojson.MarshalOptions{UseEnumNumbers: true}
					if recordJSON, merr := marshalOptions.Marshal(pbItem); merr == nil {
						msg["record"] = json.RawMessage(recordJSON)
					}
				}
			}
		}
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	this.mu.RLock()
	defer this.mu.RUnlock()

	if len(notification.AaaIds) == 0 {
		for aaaId, wc := range this.connections {
			if err := wc.writeJSON(data); err != nil {
				go this.Remove(aaaId)
			}
		}
		return
	}

	for aaaId := range notification.AaaIds {
		wc, ok := this.connections[aaaId]
		if !ok {
			continue
		}
		if err := wc.writeJSON(data); err != nil {
			go this.Remove(aaaId)
		}
	}
}

// ConnectionCount returns the number of active WebSocket connections.
func (this *WebSocketManager) ConnectionCount() int {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return len(this.connections)
}
