package integration

import (
	"fmt"

	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	ntf "github.com/saichler/l8types/go/types/l8notify"
)

const (
	IntegrationServiceName = "IntegCfg"
	IntegrationServiceArea = byte(78)
)

// Integration implements ifs.IIntegration, looking up configured integration endpoints.
// Tries the local service handler first (Notify/IntegCfg almost always run in the same
// process), falling back to a full vnic.Request round-trip otherwise.
type Integration struct {
	vnic ifs.IVNic
}

func (this *Integration) SetVNic(vnic ifs.IVNic) {
	this.vnic = vnic
}

func (this *Integration) GetIntegrationConfig(name string) (*ntf.IntegrationConfig, error) {
	filter := &ntf.IntegrationConfig{Name: name}
	if handler, ok := this.vnic.Resources().Services().ServiceHandler(IntegrationServiceName, IntegrationServiceArea); ok {
		resp := handler.Get(object.New(nil, filter), this.vnic)
		if resp.Error() != nil {
			return nil, resp.Error()
		}
		cfg, ok := resp.Element().(*ntf.IntegrationConfig)
		if !ok {
			return nil, fmt.Errorf("integration config %q not found", name)
		}
		return cfg, nil
	}
	resp := this.vnic.Request("", IntegrationServiceName, IntegrationServiceArea, ifs.GET, filter, 30)
	if resp.Error() != nil {
		return nil, resp.Error()
	}
	cfg, ok := resp.Element().(*ntf.IntegrationConfig)
	if !ok {
		return nil, fmt.Errorf("integration config %q not found", name)
	}
	return cfg, nil
}

func (this *Integration) ListIntegrationConfigs(integrationType ntf.IntegrationType) ([]*ntf.IntegrationConfig, error) {
	query := "select * from IntegrationConfig"
	if integrationType != ntf.IntegrationType_INTEGRATION_TYPE_UNSPECIFIED {
		query = fmt.Sprintf("select * from IntegrationConfig where type=%d", int32(integrationType))
	}
	var elements []interface{}
	if handler, ok := this.vnic.Resources().Services().ServiceHandler(IntegrationServiceName, IntegrationServiceArea); ok {
		elems, err := object.NewQuery(query, this.vnic.Resources())
		if err != nil {
			return nil, err
		}
		resp := handler.Get(elems, this.vnic)
		if resp.Error() != nil {
			return nil, resp.Error()
		}
		elements = resp.Elements()
	} else {
		resp := this.vnic.Request("", IntegrationServiceName, IntegrationServiceArea, ifs.GET,
			&l8api.L8Query{Text: query}, 30)
		if resp.Error() != nil {
			return nil, resp.Error()
		}
		elements = resp.Elements()
	}
	result := make([]*ntf.IntegrationConfig, 0, len(elements))
	for _, e := range elements {
		if cfg, ok := e.(*ntf.IntegrationConfig); ok {
			result = append(result, cfg)
		}
	}
	return result, nil
}
