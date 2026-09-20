package scanloop

import (
	"errors"
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

	report, err := RunTrivy(ref.RepoName, ref.Tag, ref.Digest)
	if err != nil {
		if errors.Is(err, ErrAuthRequired) {
			return authRequiredImage(ref, scommon.RegistryHost(ref.RepoName), vnic)
		}
		if errors.Is(err, ErrImageNotFound) {
			// Before writing the repo off: the tag may be gone while the
			// repo's "latest" is alive, in which case add that so the repo
			// keeps coverage (latest_fallback.go). Best-effort -- the ref
			// is marked MISSING either way.
			ensureLatestRef(ref, vnic)
			return missingImage(ref, err.Error(), vnic)
		}
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

// missingImage marks an ImageRef MISSING rather than FAILED -- the image
// reference itself couldn't be resolved (bad tag/digest, deleted from the
// registry, repo doesn't exist), as opposed to Trivy running but failing
// for some other reason. Distinct from FAILED so a user can tell "this
// image doesn't exist, fix the reference" apart from "something went wrong
// scanning a real image, retry".
func missingImage(ref *secscan.ImageRef, reason string, vnic ifs.IVNic) bool {
	ref.ScanStatus = secscan.ScanStatus_SCAN_STATUS_MISSING
	ref.ScanError = reason
	if err := l8common.PutEntity(scommon.ImageRefServiceName, scommon.ServiceArea, ref, vnic); err != nil {
		vnic.Resources().Logger().Error("scanloop: failed to mark ImageRef MISSING ", ref.ImageRefId, ": ", err.Error())
	}
	return false
}

// authRequiredImage marks an ImageRef AUTH_REQUIRED rather than FAILED --
// the registry rejected the pull for lacking (or having wrong)
// credentials, as opposed to a generic scan failure. Distinct so the
// operator can tell "the credentials mounted on this pod don't cover
// host" apart from "the scan itself broke": the fix is to widen what
// encripted/apply-registry-credentials.sh installs, not to retry.
func authRequiredImage(ref *secscan.ImageRef, host string, vnic ifs.IVNic) bool {
	ref.ScanStatus = secscan.ScanStatus_SCAN_STATUS_AUTH_REQUIRED
	ref.ScanError = "registry authentication required for " + host
	if err := l8common.PutEntity(scommon.ImageRefServiceName, scommon.ServiceArea, ref, vnic); err != nil {
		vnic.Resources().Logger().Error("scanloop: failed to mark ImageRef AUTH_REQUIRED ", ref.ImageRefId, ": ", err.Error())
	}
	return false
}
