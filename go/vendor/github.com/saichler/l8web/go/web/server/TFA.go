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

// TFA.go implements Two-Factor Authentication (2FA) HTTP endpoints.
// It provides TOTP (Time-based One-Time Password) setup and verification
// using the Layer 8 security system.
//
// TFA Flow:
// 1. User calls /tfaSetup with their user ID to receive a secret and QR code
// 2. User scans QR code with an authenticator app (Google Authenticator, Authy, etc.)
// 3. User calls /tfaSetupVerify with the TOTP code to confirm setup
// 4. Subsequent logins require the TOTP code in addition to username/password
//
// Also provides CAPTCHA generation and user registration endpoints.

package server

import (
	"net/http"

	"github.com/saichler/l8types/go/types/l8api"
	"google.golang.org/protobuf/encoding/protojson"
)

// TFASetup handles the /tfaSetup endpoint for Two-Factor Authentication setup.
// It expects a POST request with a user ID and returns a secret key and QR code
// URL that can be scanned by authenticator apps (Google Authenticator, Authy, etc.).
// The QR code encodes a TOTP URI that authenticator apps can use to generate codes.
func (this *WebService) TFASetup(w http.ResponseWriter, r *http.Request) {
	body := &l8api.L8TFASetup{}
	if !bodyToProto(w, r, "POST", body) {
		return
	}

	secret, qr, err := this.vnic.Resources().Security().TFASetup(body.UserId, this.vnic)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	resp := &l8api.L8TFASetupR{}
	resp.Secret = secret
	resp.Qr = qr
	respData, err := protojson.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respData)
}

// TFAVerify handles the /tfaVerify and /tfaSetupVerify endpoints for TOTP code verification.
// It expects a POST request with user ID, the 6-digit TOTP code, and optionally a bearer token.
// On success, it returns ok=true. This is used both for initial TFA setup verification
// and for validating TFA codes during login.
func (this *WebService) TFAVerify(w http.ResponseWriter, r *http.Request) {
	body := &l8api.L8TFAVerify{}
	if !bodyToProto(w, r, "POST", body) {
		return
	}
	authtoken, ok := this.faTokens.Load(body.UserId)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized, invalid hash"))
		return
	}
	token := authtoken.(*faTokenHash).authToken.Token
	err := this.vnic.Resources().Security().TFAVerify(body.UserId, body.Code, token, this.vnic)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	}
	this.faTokens.Delete(body.UserId)
	resp := &l8api.L8TFAVerifyR{}
	resp.Ok = true
	resp.Token = token
	respData, err := protojson.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respData)
}

// Captcha handles the /captcha endpoint for generating CAPTCHA challenges.
// It returns a CAPTCHA string that must be included in registration requests
// to prevent automated bot registrations. The CAPTCHA is typically displayed
// as an image challenge that users must solve.
func (this *WebService) Captcha(w http.ResponseWriter, r *http.Request) {
	cp := this.vnic.Resources().Security().Captcha()
	resp := &l8api.Captcha{}
	resp.Captcha = cp

	respData, err := protojson.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respData)
}

// Register handles the /register endpoint for new user registration.
// It expects a POST request with username, password, and a valid CAPTCHA response.
// The CAPTCHA must match one previously obtained from the /captcha endpoint.
// Returns HTTP 200 on success or HTTP 401 if registration fails (invalid CAPTCHA,
// duplicate user, etc.).
func (this *WebService) Register(w http.ResponseWriter, r *http.Request) {
	body := &l8api.AuthUser{}
	if !bodyToProto(w, r, "POST", body) {
		return
	}
	err := this.vnic.Resources().Security().Register(body.User, body.Pass, body.Captcha, this.vnic)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}
