package scanjob

import (
	"strings"
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
	refs := make([]*secscan.ImageRef, 0, len(job.ImageRefIds))
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
		refs = append(refs, ref)
	}

	// An image over MaxAutoScanBytes is dropped from a job that has other
	// images in it, and kept when it is the only one. The backstop lives
	// here rather than only in the UI so it holds for any client, including
	// a direct POST -- the UIs filter too, so in practice this rarely
	// fires.
	//
	// Dropped, not rejected: one oversized image should not block a sweep
	// of fifty others. It is named in skippedOversized so the caller can
	// say what happened; nothing is silently discarded.
	var skipped []string
	if len(refs) > 1 {
		kept := make([]string, 0, len(refs))
		for _, ref := range refs {
			if ref.SizeBytes > scommon.MaxAutoScanBytes {
				skipped = append(skipped, ref.RepoName+":"+ref.Tag)
				continue
			}
			kept = append(kept, ref.ImageRefId)
		}
		// Every image in the request was oversized -- there is no job to
		// run, and silently returning an empty one would look like a scan
		// that found nothing.
		if len(kept) == 0 {
			return object.NewError("every image in this request is over " +
				scommon.HumanBytes(scommon.MaxAutoScanBytes) +
				"; scan each one on its own: " + strings.Join(skipped, ", "))
		}
		if len(skipped) > 0 {
			// Logged rather than returned on the ScanJob: that would be
			// transient response-only state on a persisted entity. Both
			// UIs filter before POSTing and so already know what they
			// left out; this is for direct API callers and for anyone
			// reading the logs afterwards.
			vnic.Resources().Logger().Info("scanjob: skipped ", len(skipped),
				" image(s) over ", scommon.HumanBytes(scommon.MaxAutoScanBytes),
				" in a multi-image job: ", strings.Join(skipped, ", "))
		}
		job.ImageRefIds = kept
		job.TotalImages = int32(len(kept))
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
