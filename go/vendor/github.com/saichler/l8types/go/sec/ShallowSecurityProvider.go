package sec

import (
	"crypto/md5"
	"encoding/base64"
	"errors"
	"github.com/saichler/l8types/go/aes"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/nets"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8sysconfig"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ShallowSecurityProvider implements ISecurityProvider with basic AES encryption.
// Uses a hardcoded secret for key derivation - suitable for testing only.
type ShallowSecurityProvider struct {
	secret string
	key    string
}

// NewShallowSecurityProvider creates a new provider with a hardcoded secret.
func NewShallowSecurityProvider() *ShallowSecurityProvider {
	sp := &ShallowSecurityProvider{}
	hash := md5.New()
	secret := "Shallow Security Provider"
	hash.Write([]byte(secret))
	kHash := hash.Sum(nil)
	sp.key = base64.StdEncoding.EncodeToString(kHash)
	sp.secret = secret
	return sp
}

// CanDial establishes a TCP connection to the specified host and port.
func (this *ShallowSecurityProvider) CanDial(host string, port uint32) (net.Conn, error) {
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return net.Dial("tcp", host+":"+strconv.Itoa(int(port)))
}

// CanAccept always allows incoming connections (permissive).
func (this *ShallowSecurityProvider) CanAccept(conn net.Conn) error {
	return nil
}

// ValidateConnection verifies the connection by exchanging encrypted secrets.
func (this *ShallowSecurityProvider) ValidateConnection(conn net.Conn, config *l8sysconfig.L8SysConfig) error {
	err := nets.WriteEncrypted(conn, []byte(this.secret), config, this)
	if err != nil {
		conn.Close()
		return err
	}

	secret, err := nets.ReadEncrypted(conn, config, this)
	if err != nil {
		conn.Close()
		return err
	}

	if this.secret != secret {
		conn.Close()
		return errors.New("incorrect Secret/Key, aborting connection")
	}

	return nets.ExecuteProtocol(conn, config, this)
}

// Encrypt encrypts data using AES with the derived key.
func (this *ShallowSecurityProvider) Encrypt(data []byte) (string, error) {
	return aes.Encrypt(data, this.key)
}

// Decrypt decrypts AES-encrypted data using the derived key.
func (this *ShallowSecurityProvider) Decrypt(data string) ([]byte, error) {
	return aes.Decrypt(data, this.key)
}

// CanDoAction always permits any action (permissive authorization).
func (this *ShallowSecurityProvider) CanDoAction(vnic ifs.IVNic, action ifs.Action, o ifs.IElements, uuid string, token string, salts ...string) error {
	return nil
}

// ScopeView returns the original elements without filtering (permissive).
func (this *ShallowSecurityProvider) ScopeView(vnic ifs.IVNic, o ifs.IElements, uuid string, token string, salts ...string) ifs.IElements {
	return o
}

// ScopeView returns the original interface without filtering (permissive).
func (this *ShallowSecurityProvider) ScopeItem(r ifs.IResources, o interface{}, uuid string, token string, salts ...string) interface{} {
	return o
}

// AllowedTypes returns nil (permissive - all types allowed).
func (this *ShallowSecurityProvider) AllowedTypes(vnic ifs.IVNic, token string) []string {
	return nil
}

// AllowedActions returns nil (permissive - all actions allowed).
func (this *ShallowSecurityProvider) AllowedActions(vnic ifs.IVNic, token string) map[string][]int32 {
	return nil
}

// Authenticate always succeeds with a dummy bearer token (testing only).
func (this *ShallowSecurityProvider) Authenticate(user string, pass string, vnic ifs.IVNic) *l8api.AuthToken {
	return &l8api.AuthToken{Token: "bearer token"}
}

// ValidateToken always validates tokens successfully (permissive).
func (this *ShallowSecurityProvider) ValidateToken(token string, vnic ifs.IVNic) (string, bool) {
	return ifs.NewUuid(), true
}
func (this *ShallowSecurityProvider) Message(aaaid string, vnic ifs.IVNic) (*ifs.Message, error) {
	return &ifs.Message{}, nil
}

func (this *ShallowSecurityProvider) TFASetup(userid string, nic ifs.IVNic) (string, []byte, error) {
	return "", nil, nil
}
func (this *ShallowSecurityProvider) TFAVerify(userid string, code string, bearer string, nic ifs.IVNic) error {
	return nil
}
func (this *ShallowSecurityProvider) Captcha() []byte {
	return nil
}
func (this *ShallowSecurityProvider) Register(userId, password, captcha string, vnic ifs.IVNic) error {
	return nil
}

func (this *ShallowSecurityProvider) RequestPasswordReset(userIdOrEmail, captcha, resetBaseURL string, vnic ifs.IVNic) error {
	return nil
}

func (this *ShallowSecurityProvider) ResetPassword(userId, token, newPassword string, vnic ifs.IVNic) error {
	return nil
}

func (this *ShallowSecurityProvider) Credential(crId, cId string, r ifs.IResources) (string, string, string, string, error) {
	if crId == "sim" {
		if cId == "ssh" {
			return "simadmin", "simadmin", "simadmin", "simadmin", nil
		}
		if cId == "snmp" {
			return "public", "private", "", "", nil
		}
		if crId == "rest" {
			return "admin", "admin", "", "", nil
		}
		if crId == "graph" {
			return "admin", "admin", "", "", nil
		}
	}
	return "admin", "admin", "admin", "5432", nil
}

func (this *ShallowSecurityProvider) AddAdjacent(ifs.ISecurityProvider) {}

func (this *ShallowSecurityProvider) NewSystemConfig() *l8sysconfig.L8SysConfig {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	dataDirectory := filepath.Join(homeDir, "data")
	os.MkdirAll(dataDirectory, 0755)
	sysconfig := &l8sysconfig.L8SysConfig{MaxDataSize: 1024 * 1024 * 50,
		RxQueueSize:              100000,
		TxQueueSize:              100000,
		VnetPort:                 uint32(10005),
		KeepAliveIntervalSeconds: 30,
		LogConfig:                &l8sysconfig.L8LogConfig{LogDirectory: "", VnetPort: 10010},
		DataStoreConfig:          &l8sysconfig.L8DataStoreConfig{Type: "postgres", Name: "admin"},
		TimeSeriesStoreConfig:    &l8sysconfig.L8DataStoreConfig{Type: "postgres", Name: "admints"},
		WebConfig:                &l8sysconfig.L8WebAppConfig{WebPort: 4443, EndPointPrefix: "web"},
		DataDirectory:            dataDirectory,
	}
	domain, privat, public := CreateCertBundle()
	sysconfig.WebConfig.DomainCertPem = domain
	sysconfig.WebConfig.PrivateKeyPem = privat
	sysconfig.WebConfig.PublicKeyPem = public
	return sysconfig
}
