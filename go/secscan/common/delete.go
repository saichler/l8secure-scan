package common

import (
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

// DeleteEntitiesByQuery deletes every entity matching an L8QL query. Not
// provided by l8common/go/common (which has GetEntitiesByQuery/PutEntity/
// PostEntity but no Delete equivalent) -- mirrors the same hasLocal/
// vnic.Request fallback shape as those real helpers, and the same
// handler.Delete(elems, vnic) call verified against
// ../l8alarms/go/alm/archiving/engine.go's deleteAlarm (the only real
// ecosystem precedent for IServiceHandler.Delete usage).
func DeleteEntitiesByQuery(serviceName string, serviceArea byte, query string, vnic ifs.IVNic) error {
	handler, ok := vnic.Resources().Services().ServiceHandler(serviceName, serviceArea)
	if ok {
		elems, err := object.NewQuery(query, vnic.Resources())
		if err != nil {
			return err
		}
		resp := handler.Delete(elems, vnic)
		return resp.Error()
	}
	resp := vnic.Request("", serviceName, serviceArea, ifs.DELETE, query, 30)
	return resp.Error()
}
