package sec

import (
	"errors"
	"github.com/saichler/l8types/go/ifs"
	"plugin"
)

// LoadSecurityProvider loads the security provider plugin from /var/loader.so.
// This function uses Go's plugin system to dynamically load the security implementation.
func LoadSecurityProvider(args ...interface{}) (ifs.ISecurityProvider, error) {
	loaderFile, err := plugin.Open("/var/loader.so")
	if err != nil {
		return nil, errors.New("Failed to load security provider #1: " + err.Error())
	}
	loader, err := loaderFile.Lookup("Loader")
	if err != nil {
		return nil, errors.New("Failed to load security provider #2: " + err.Error())
	}
	if loader == nil {
		return nil, errors.New("Failed to load security provider #3: Nil Loader")
	}
	loaderInterface := *loader.(*ifs.ISecurityProviderLoader)
	securityLoader := loaderInterface.(ifs.ISecurityProviderLoader).(ifs.ISecurityProviderLoader)
	return securityLoader.LoadSecurityProvider(args...)
}
