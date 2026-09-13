// Package scanloop is the Trivy scan loop (PRD §13.1): polls for QUEUED
// ScanJobs, claims one, and runs Trivy against each of its images.
package scanloop

import (
	"sync"
	"time"

	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/secscan/scanner/pollworker"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/workers"
)

const (
	pollInterval  = 10 * time.Second
	jobPoolSize   = 2 // claimed ScanJobs processed concurrently
	imagePoolSize = 4 // images within one job processed concurrently
)

// Run starts the scan loop and blocks until stop is closed.
func Run(vnic ifs.IVNic, stop <-chan struct{}) {
	pollworker.Run(pollworker.Config{
		ServiceName: scommon.ScanJobServiceName,
		ServiceArea: scommon.ServiceArea,
		// 1 = JOB_STATUS_QUEUED. L8Query enum comparisons are bare
		// integers, never quoted names (see PRD §13/plans/PROGRESS.md).
		Query:    "select * from ScanJob where status=1",
		Interval: pollInterval,
		PoolSize: jobPoolSize,
		Claim:    claim,
		Work:     work,
	}, vnic, stop)
}

// claim is the QUEUED->RUNNING transition. No compare-and-swap exists on
// PUT (verified, PRD §13's concurrency note) -- single replica for v1
// makes an unconditional PUT safe; this is an already-accepted design
// decision, not re-litigated here.
func claim(item interface{}, vnic ifs.IVNic) bool {
	job, ok := item.(*secscan.ScanJob)
	if !ok || job == nil {
		return false
	}
	job.Status = secscan.JobStatus_JOB_STATUS_RUNNING
	if err := l8common.PutEntity(scommon.ScanJobServiceName, scommon.ServiceArea, job, vnic); err != nil {
		vnic.Resources().Logger().Error("scanloop: failed to claim ScanJob ", job.ScanJobId, ": ", err.Error())
		return false
	}
	return true
}

// work fans out across the job's images (PRD §13.1 step 1: "worker pool,
// bounded concurrency, this is the harness's work function" -- a second,
// inner level of bounded concurrency, separate from the outer harness's
// per-job pool), tallies the outcome, and finalizes the ScanJob.
func work(item interface{}, vnic ifs.IVNic) {
	job, ok := item.(*secscan.ScanJob)
	if !ok || job == nil {
		return
	}

	var mu sync.Mutex
	var completed, failed int32

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
				completed++
			} else {
				failed++
			}
			mu.Unlock()
		}))
	}
	wg.Wait()

	job.CompletedImages = completed
	job.FailedImages = failed
	switch {
	case failed == 0:
		job.Status = secscan.JobStatus_JOB_STATUS_COMPLETED
	case completed == 0:
		job.Status = secscan.JobStatus_JOB_STATUS_FAILED
	default:
		job.Status = secscan.JobStatus_JOB_STATUS_PARTIAL
	}
	job.CompletedAt = time.Now().Unix()

	if err := l8common.PutEntity(scommon.ScanJobServiceName, scommon.ServiceArea, job, vnic); err != nil {
		vnic.Resources().Logger().Error("scanloop: failed to finalize ScanJob ", job.ScanJobId, ": ", err.Error())
	}
}

type runFunc func()

func (f runFunc) Run() { f() }
