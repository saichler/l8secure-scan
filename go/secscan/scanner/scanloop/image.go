package scanloop

import (
	"time"

	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

// scanOneImage is the per-image unit of work (PRD §13.1 step 1-3).
// Returns true on a successful scan, false on any failure (fetch, Trivy,
// parse, or write) -- the caller tallies these into ScanJob.completed/
// failedImages.
func scanOneImage(refId string, vnic ifs.IVNic) bool {
	result, err := l8common.GetEntity(scommon.ImageRefServiceName, scommon.ServiceArea, &secscan.ImageRef{ImageRefId: refId}, vnic)
	if err != nil {
		vnic.Resources().Logger().Error("scanloop: failed to fetch ImageRef ", refId, ": ", err.Error())
		return false
	}
	ref, ok := result.(*secscan.ImageRef)
	if !ok || ref == nil {
		vnic.Resources().Logger().Error("scanloop: ImageRef not found: ", refId)
		return false
	}

	ref.ScanStatus = secscan.ScanStatus_SCAN_STATUS_SCANNING
	if err := l8common.PutEntity(scommon.ImageRefServiceName, scommon.ServiceArea, ref, vnic); err != nil {
		vnic.Resources().Logger().Error("scanloop: failed to mark ImageRef SCANNING ", refId, ": ", err.Error())
		return false
	}

	report, err := runTrivy(ref.RepoName, ref.Tag, ref.Digest)
	if err != nil {
		return failImage(ref, err.Error(), vnic)
	}

	if err := replaceFindings(ref, report, vnic); err != nil {
		return failImage(ref, err.Error(), vnic)
	}

	total, distinct := countSeverities(report)
	ref.ScanStatus = secscan.ScanStatus_SCAN_STATUS_COMPLETED
	ref.TotalCounts = total
	ref.DistinctCounts = distinct
	ref.LastScannedAt = time.Now().Unix()
	ref.ScanError = ""
	if err := l8common.PutEntity(scommon.ImageRefServiceName, scommon.ServiceArea, ref, vnic); err != nil {
		vnic.Resources().Logger().Error("scanloop: failed to complete ImageRef ", refId, ": ", err.Error())
		return false
	}
	return true
}

func failImage(ref *secscan.ImageRef, reason string, vnic ifs.IVNic) bool {
	ref.ScanStatus = secscan.ScanStatus_SCAN_STATUS_FAILED
	ref.ScanError = reason
	if err := l8common.PutEntity(scommon.ImageRefServiceName, scommon.ServiceArea, ref, vnic); err != nil {
		vnic.Resources().Logger().Error("scanloop: failed to mark ImageRef FAILED ", ref.ImageRefId, ": ", err.Error())
	}
	return false
}
