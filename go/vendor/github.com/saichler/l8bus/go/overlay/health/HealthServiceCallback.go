package health

import (
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8health"
	"github.com/saichler/l8utils/go/utils/logger"
)

type HealthServiceCallback struct{}

func (this *HealthServiceCallback) Before(any interface{}, action ifs.Action, notify bool, vnic ifs.IVNic) (interface{}, bool, error) {
	return nil, true, nil
}

func (this *HealthServiceCallback) After(any interface{}, action ifs.Action, notify bool, vnic ifs.IVNic) (interface{}, bool, error) {
	health, ok := any.(*l8health.L8Health)
	if action == ifs.GET && ok {
		if health.PprofCollect {
			health.PprofMemory, health.PprofCpu, _ = logger.DumpPprofToBytes()
		}
		return health, true, nil
	}
	return nil, true, nil
}
