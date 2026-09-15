package scanjob

import (
	"time"

	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/secscan/scanner/scanloop"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

// Post validates the request (absorbed verbatim from the old
// ScanJobServiceCallback.beforePostScanJob -- ScanJobs itself has no
// ServiceCallback anymore, plans/scanjob-live-progress.md Phase 2), persists
// the initial job to ScanJobs, and kicks off scanning in the background. The
// HTTP response returns immediately with the created job; scanning continues
// after (MainPackageMinimal -- no long-running work synchronously in the
// handler).
func (this *ScanJobHandler) Post(elems ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	job, ok := elems.Element().(*secscan.ScanJob)
	if !ok {
		return object.NewError("invalid ScanJob type")
	}
	if job.CustomerId == "" {
		return object.NewError("customerId is required")
	}
	if len(job.ImageRefIds) == 0 {
		return object.NewError("imageRefIds is required")
	}
	for _, refId := range job.ImageRefIds {
		result, err := l8common.GetEntity(scommon.ImageRefServiceName, scommon.ServiceArea, &secscan.ImageRef{ImageRefId: refId}, vnic)
		if err != nil {
			return object.NewError(err.Error())
		}
		ref, ok := result.(*secscan.ImageRef)
		if !ok || ref == nil {
			return object.NewError("imageRefId " + refId + " not found")
		}
		if ref.CustomerId != job.CustomerId {
			return object.NewError("imageRefId " + refId + " does not belong to customer " + job.CustomerId)
		}
	}

	l8common.GenerateID(&job.ScanJobId)
	// No JOB_STATUS_QUEUED -- with no poll/claim step, a job goes straight to
	// RUNNING the moment it's created (plans/scanjob-live-progress.md Decision 2).
	job.Status = secscan.JobStatus_JOB_STATUS_RUNNING
	job.TotalImages = int32(len(job.ImageRefIds))
	job.RequestedAt = time.Now().Unix()

	if _, err := l8common.PostEntity(scommon.ScanJobsServiceName, scommon.ServiceArea, job, vnic); err != nil {
		return object.NewError(err.Error())
	}

	go scanloop.Run(job, vnic)

	return object.New(nil, job)
}
