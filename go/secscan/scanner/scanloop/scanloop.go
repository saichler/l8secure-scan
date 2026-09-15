// Package scanloop scans a ScanJob's images directly (plans/scanjob-live-progress.md
// Phase 3) -- no poll/claim; invoked synchronously from the stateless scanjob
// service's background goroutine right after the job is created.
package scanloop

import (
	"sync"
	"time"

	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/workers"
)

// imagePoolSize bounds concurrency across one job's images.
const imagePoolSize = 4

// Run scans every image in job. Fans out across job's images with bounded
// concurrency (PRD §13.1 step 1), and persists job's progress to ScanJobs
// after every single image finishes -- not just once at the end -- so the
// live progress bar has something to react to as scanning happens
// (plans/scanjob-live-progress.md). PUT is a full-record replace, so the
// counter-update-then-PUT sequence is serialized under mu even though the
// scanning itself stays concurrent (Decision 3).
func Run(job *secscan.ScanJob, vnic ifs.IVNic) {
	var mu sync.Mutex

	pool := workers.NewWorkers(imagePoolSize)
	var wg sync.WaitGroup
	for _, refId := range job.ImageRefIds {
		refId := refId
		wg.Add(1)
		pool.Run(runFunc(func() {
			defer wg.Done()
			ok := scanOneImage(refId, vnic)

			mu.Lock()
			if ok {
				job.CompletedImages++
			} else {
				job.FailedImages++
			}
			if err := l8common.PutEntity(scommon.ScanJobsServiceName, scommon.ServiceArea, job, vnic); err != nil {
				vnic.Resources().Logger().Error("scanloop: failed to update ScanJob progress ", job.ScanJobId, ": ", err.Error())
			}
			mu.Unlock()
		}))
	}
	wg.Wait()

	switch {
	case job.FailedImages == 0:
		job.Status = secscan.JobStatus_JOB_STATUS_COMPLETED
	case job.CompletedImages == 0:
		job.Status = secscan.JobStatus_JOB_STATUS_FAILED
	default:
		job.Status = secscan.JobStatus_JOB_STATUS_PARTIAL
	}
	job.CompletedAt = time.Now().Unix()

	if err := l8common.PutEntity(scommon.ScanJobsServiceName, scommon.ServiceArea, job, vnic); err != nil {
		vnic.Resources().Logger().Error("scanloop: failed to finalize ScanJob ", job.ScanJobId, ": ", err.Error())
	}
}

type runFunc func()

func (f runFunc) Run() { f() }
