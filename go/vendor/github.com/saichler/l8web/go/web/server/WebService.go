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

// WebService.go implements the core web service manager for Layer 8.
// It handles service activation, authentication endpoints, user registration,
// Two-Factor Authentication (TFA) setup, and CAPTCHA generation.
//
// Built-in HTTP endpoints registered by this service:
//   - /auth         - User authentication (returns bearer token)
//   - /registry     - Type registry access
//   - /tfaSetup     - Two-Factor Authentication setup (returns QR code)
//   - /tfaSetupVerify - TFA verification
//   - /tfaVerify    - TFA code verification during login
//   - /captcha      - CAPTCHA challenge generation
//   - /register     - User registration with CAPTCHA
//   - /permissions  - Per-type allowed actions for the authenticated user

package server

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8bus/go/overlay/plugins"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8notify"
	"github.com/saichler/l8types/go/types/l8web"
	"github.com/saichler/l8utils/go/utils/web"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	// ServiceTypeName is the identifier used when registering the WebService
	// with the Layer 8 service manager.
	ServiceTypeName = "WebService"
)

// WebService implements the Layer 8 service handler interface for web service
// management. It handles service activation, HTTP endpoint registration, and
// cross-VNet authentication token mapping.
type WebService struct {
	server    ifs.IWebServer // The REST server instance
	vnic      ifs.IVNic      // Primary VNic for service communication
	adjacents []ifs.IVNic    // Adjacent VNet Vnic for cross-network auth
	faTokens  *sync.Map
	wsManager *WebSocketManager
}

type faTokenHash struct {
	authToken *l8api.AuthToken
	hash      string
}

// mtx provides thread-safe access to shared registration state.
var mtx = &sync.Mutex{}

// registered tracks VNet ports that have already been registered to prevent duplicates.
var registered = map[uint32]bool{}

// registeredAuth tracks whether authentication endpoints have been registered.
var registeredAuth = false

// authEnabled indicates whether bearer token authentication is globally enabled.
var authEnabled = false

// proxyMode indicates whether the server is running behind a reverse proxy.
var proxyMode = false

// Activate initializes the WebService and registers all HTTP endpoints.
// It sets up authentication, TFA, CAPTCHA, and registration handlers.
// If additional VNic instances are provided in the SLA args, they are
// registered as adjacent networks for cross-VNet authentication.
func (this *WebService) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	this.vnic = vnic
	this.faTokens = &sync.Map{}
	vnic.Resources().Registry().Register(&l8web.L8WebService{})
	this.server = sla.Args()[0].(ifs.IWebServer)
	go func() {
		time.Sleep(time.Second * 2)
		fmt.Println("Sending Get Multicast for EndPoints ", vnic.Resources().SysConfig().VnetPort)
		vnic.Multicast(health.ServiceName, 0, ifs.EndPoints, nil)
	}()

	mtx.Lock()
	defer mtx.Unlock()

	if !registeredAuth {
		registeredAuth = true
		if len(sla.Args()) > 1 {
			proxy, ok := sla.Args()[1].(ifs.IWebProxy)
			if ok {
				proxy.SetValidator(this)
				proxy.RegisterHandlers(nil)
			}
		}
		http.DefaultServeMux.HandleFunc("/auth", this.Auth)
		http.DefaultServeMux.HandleFunc("/registry", this.Registry)
		http.DefaultServeMux.HandleFunc("/tfaSetup", this.TFASetup)
		http.DefaultServeMux.HandleFunc("/tfaSetupVerify", this.TFAVerify)
		http.DefaultServeMux.HandleFunc("/tfaVerify", this.TFAVerify)
		http.DefaultServeMux.HandleFunc("/captcha", this.Captcha)
		http.DefaultServeMux.HandleFunc("/register", this.Register)
		http.DefaultServeMux.HandleFunc("/permissions", this.Permissions)

		this.wsManager = NewWebSocketManager(vnic)
		http.DefaultServeMux.HandleFunc("/ws", this.wsManager.HandleUpgrade)

		wsNotifySvc := &WsNotifyService{}
		wsSla := ifs.NewServiceLevelAgreement(wsNotifySvc, WsNotifyServiceName, WsNotifyServiceArea, false, nil)
		wsSla.SetServiceItem(&l8notify.L8NotificationSet{})
		wsSla.SetArgs(this.wsManager)
		vnic.Resources().Services().Activate(wsSla, vnic)
	}

	for _, n := range sla.Args() {
		nic, ok := n.(ifs.IVNic)
		if ok {
			_, ok = registered[nic.Resources().SysConfig().VnetPort]
			if !ok {
				if this.adjacents == nil {
					this.adjacents = make([]ifs.IVNic, 0)
				}
				this.adjacents = append(this.adjacents, nic)
				this.vnic.Resources().Security().AddAdjacent(nic.Resources().Security())
				registered[nic.Resources().SysConfig().VnetPort] = true
				go func() {
					time.Sleep(time.Second * 5)
					nic.Resources().Services().Activate(sla, nic)
				}()
			}
		}
	}
	return nil
}

// Auth handles user authentication requests at the /auth endpoint.
// It expects a POST request with JSON body containing user and pass fields.
// On successful authentication, it returns a bearer token and sets an HTTP-only
// cookie for browser-based clients. Also handles TFA status (needTfa, setupTfa).
// For cross-VNet setups, it also authenticates with adjacent networks and maps tokens.
func (this *WebService) Auth(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to read user/pass #1"))
		w.Write([]byte(err.Error()))
		fmt.Println("Failed to read user/pass #1")
		return
	}
	user := &l8api.AuthUser{}
	err = protojson.Unmarshal(data, user)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to read user/pass #2"))
		w.Write([]byte(err.Error()))
		fmt.Println("Failed to read user/pass #2")
		return
	}

	pending, ok := this.faTokens.Load(user.User)
	if ok {
		this.faTokens.Delete(user.User)
		faPending := pending.(*faTokenHash)
		if faPending.authToken.TokenHash != user.TokenHash {
			w.WriteHeader(http.StatusUnauthorized)
			authToken := &l8api.AuthToken{}
			authToken.Error = "Mismatch Hash"
			jsn, _ := protojson.Marshal(authToken)
			w.Write(jsn)
			fmt.Println("Failed to authenticate hash #4")
			return
		}
		jsn, _ := protojson.Marshal(faPending.authToken)
		w.WriteHeader(http.StatusOK)
		w.Write(jsn)
	}

	authToken := this.vnic.Resources().Security().Authenticate(user.User, user.Pass, this.vnic)
	if authToken.Error != "" {
		w.WriteHeader(http.StatusUnauthorized)
		jsn, _ := protojson.Marshal(authToken)
		w.Write(jsn)
		this.vnic.Resources().Logger().Warning("Failed to authenticate user/pass #3")
		return
	}

	if authToken.NeedTfa {
		fa := &faTokenHash{authToken: authToken, hash: authToken.TokenHash}
		this.faTokens.Store(user.User, fa)

		faToken := &l8api.AuthToken{}
		faToken.NeedTfa = authToken.NeedTfa
		faToken.SetupTfa = authToken.SetupTfa
		faToken.TokenHash = authToken.TokenHash
		jsn, _ := protojson.Marshal(faToken)
		w.WriteHeader(http.StatusOK)
		w.Write(jsn)
		return
	}

	jsn, _ := protojson.Marshal(authToken)
	http.SetCookie(w, &http.Cookie{
		Name:     BearerCookieName,
		Value:    authToken.Token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		Secure:   true, // false for local dev without HTTPS
		SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusOK)
	w.Write(jsn)
}

// DeActivate performs cleanup when the service is being shut down.
// Currently a no-op as cleanup is handled elsewhere.
func (this *WebService) DeActivate() error {
	return nil
}

// Post handles incoming web service registration requests via Layer 8 messaging.
// When a new web service is discovered in the network, this method deserializes
// the service definition, loads any associated plugins, and registers the service
// with the local REST server.
func (this *WebService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	webService := pb.Element().(*l8web.L8WebService)
	ws := web.New(webService.ServiceName, byte(webService.ServiceArea), webService.Vnet)
	err := ws.DeSerialize(webService, this.vnic.Resources().Registry())
	if err != nil {
		vnic.Resources().Logger().Error(err.Error())
	}
	vnic.Resources().Logger().Info("Received Webservice ", ws.ServiceName(), " ", ws.ServiceArea())
	if ws.Plugin() != "" {
		plg := &l8web.L8Plugin{Data: ws.Plugin()}
		plugins.LoadPlugin(plg, vnic)
	}
	this.server.RegisterWebService(ws, vnic)
	return object.New(nil, nil)
}

// Put handles PUT requests for the WebService. Not implemented.
func (this *WebService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Patch handles PATCH requests for the WebService. Not implemented.
func (this *WebService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Delete handles DELETE requests for the WebService. Not implemented.
func (this *WebService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// GetCopy handles copy GET requests for the WebService. Not implemented.
func (this *WebService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Get handles GET requests for the WebService. Returns an empty response.
func (this *WebService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return object.New(nil, nil)
}

// Failed handles failed requests for the WebService. Not implemented.
func (this *WebService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}

// TransactionConfig returns the transaction configuration for this service.
// Returns nil as WebService doesn't use transactions.
func (this *WebService) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

// WebService returns the web service interface. Returns nil as this is the manager.
func (this *WebService) WebService() ifs.IWebService {
	return nil
}

// Registry handles requests to the /registry endpoint, returning the type
// registry as JSON. Requires authentication if globally enabled.
func (this *WebService) Registry(w http.ResponseWriter, r *http.Request) {
	if authEnabled {
		bearer := r.Header.Get("Authorization")
		if bearer == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, ok := this.vnic.Resources().Security().ValidateToken(bearer, this.vnic)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	typeList := this.vnic.Resources().Registry().TypeList()
	byt, _ := protojson.Marshal(typeList)
	w.WriteHeader(http.StatusOK)
	w.Write(byt)
}

// Permissions handles requests to the /permissions endpoint, returning the
// per-type allowed actions for the authenticated user as JSON.
// Response format: { "TypeName": [1,2,5], ... } where 1=POST,2=PUT,3=PATCH,4=DELETE,5=GET
func (this *WebService) Permissions(w http.ResponseWriter, r *http.Request) {
	bearer := r.Header.Get("Authorization")
	if bearer == "" {
		bearer = extractToken(r)
	}
	if bearer == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	aaaid, ok := this.vnic.Resources().Security().ValidateToken(bearer, this.vnic)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	actions := this.vnic.Resources().Security().AllowedActions(this.vnic, aaaid)
	if actions == nil {
		// nil means permissive (no security provider or shallow provider) — return empty map
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
		return
	}
	// Serialize as JSON manually for map[string][]int32
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{"))
	first := true
	for typeName, actionList := range actions {
		if !first {
			w.Write([]byte(","))
		}
		first = false
		w.Write([]byte(fmt.Sprintf("%q:[", typeName)))
		for i, a := range actionList {
			if i > 0 {
				w.Write([]byte(","))
			}
			w.Write([]byte(fmt.Sprintf("%d", a)))
		}
		w.Write([]byte("]"))
	}
	w.Write([]byte("}"))
}

// ValidateBearerToken validates the bearer token from an HTTP request.
// It first checks the Authorization header, then falls back to extractToken
// (which checks cookies and query parameters). Returns an error if the token
// is missing or invalid. This method is used by the reverse proxy for
// protected endpoint validation.
func (this *WebService) ValidateBearerToken(r *http.Request) error {
	bearer := r.Header.Get("Authorization")
	if bearer == "" {
		bearer = extractToken(r)
	}
	if bearer == "" {
		return errors.New("unauthorized")
	}
	_, ok := this.vnic.Resources().Security().ValidateToken(bearer, this.vnic)
	if !ok {
		return errors.New("unauthorized")
	}
	return nil
}
