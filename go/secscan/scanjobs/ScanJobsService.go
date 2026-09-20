// Package scanjobs is the ORM/Postgres-backed persistence service for
// ScanJob records. It does nothing but persist -- no ServiceCallback, no
// validation, no ID generation, nothing polls it. All of that now lives in
// the stateless secscan/scanjob action service, which is the only writer
// (plans/scanjob-live-progress.md).
package scanjobs

import (
	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

func Activate(creds, dbname string, vnic ifs.IVNic) {
	sla := l8common.NewOrmSLA(scommon.ScanJobsServiceName, scommon.ServiceArea, "ScanJobId", nil,
		&secscan.ScanJob{}, &secscan.ScanJobList{})
	l8common.ActivateService(sla, creds, dbname, vnic)
}
