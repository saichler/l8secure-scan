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
	l8common.ActivateService(l8common.ServiceConfig{
		ServiceName: scommon.ScanJobsServiceName, ServiceArea: scommon.ServiceArea,
		PrimaryKey: "ScanJobId",
	}, &secscan.ScanJob{}, &secscan.ScanJobList{}, creds, dbname, vnic)
}
