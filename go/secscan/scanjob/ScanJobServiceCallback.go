package scanjob

import (
	"errors"
	"time"

	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

// PRD §9: Before() on POST -- takes customer_id from the request body,
// validates every image_ref_ids entry actually belongs to that customer_id
// (a data-integrity sanity check on the request's own internal
// consistency, not a security boundary -- customer_id itself isn't
// independently verified against caller identity, §4), auto-generates the
// primary key, and sets QUEUED/totalImages/requestedAt defaults. No
// scanning happens inline (MainPackageMinimal) -- secscan-scanner polls
// for queued jobs (§13).
func newScanJobServiceCallback() ifs.IServiceCallback {
	return l8common.NewServiceCallback(
		"ScanJob",
		func(v interface{}) bool { _, ok := v.(*secscan.ScanJob); return ok },
		func(v interface{}) {},
		nil,
		beforePostScanJob,
	)
}

func beforePostScanJob(e interface{}, action ifs.Action, vnic ifs.IVNic) error {
	if action != ifs.POST {
		return nil
	}
	job, ok := e.(*secscan.ScanJob)
	if !ok {
		return errors.New("invalid ScanJob type")
	}
	if job.CustomerId == "" {
		return errors.New("customerId is required")
	}
	if len(job.ImageRefIds) == 0 {
		return errors.New("imageRefIds is required")
	}
	for _, refId := range job.ImageRefIds {
		result, err := l8common.GetEntity(scommon.ImageRefServiceName, scommon.ServiceArea, &secscan.ImageRef{ImageRefId: refId}, vnic)
		if err != nil {
			return err
		}
		ref, ok := result.(*secscan.ImageRef)
		if !ok || ref == nil {
			return errors.New("imageRefId " + refId + " not found")
		}
		if ref.CustomerId != job.CustomerId {
			return errors.New("imageRefId " + refId + " does not belong to customer " + job.CustomerId)
		}
	}

	l8common.GenerateID(&job.ScanJobId)
	job.Status = secscan.JobStatus_JOB_STATUS_QUEUED
	job.TotalImages = int32(len(job.ImageRefIds))
	job.RequestedAt = time.Now().Unix()
	return nil
}
