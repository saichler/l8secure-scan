// Package pollworker is the one shared poll-claim-dispatch harness (PRD
// §13) used by both loops secscan-scanner runs -- the metadata resolver
// loop and the Trivy scan loop -- so the "poll a table on an interval,
// claim matched rows, dispatch to a bounded worker pool" behavior exists
// exactly once (Duplication Prevention, Second Instance Rule).
package pollworker

import (
	"sync"
	"time"

	l8common "github.com/saichler/l8common/go/common"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/workers"
)

// ClaimFunc decides whether a matched row should be processed and, if so,
// performs any state transition needed to claim it (e.g. QUEUED->RUNNING).
// nil means every matched row is processed (an idempotent, safe-to-retry
// lookup needs no exclusive claim -- the resolver loop).
type ClaimFunc func(item interface{}, vnic ifs.IVNic) bool

// WorkFunc processes one claimed row.
type WorkFunc func(item interface{}, vnic ifs.IVNic)

// Config configures one loop.
type Config struct {
	ServiceName string
	ServiceArea byte
	Query       string
	Interval    time.Duration
	PoolSize    int
	Claim       ClaimFunc
	Work        WorkFunc
}

// Run ticks Config.Query on Config.Interval until stop is closed, running
// once immediately on entry. Each tick's matched rows are dispatched to a
// pool bounded at Config.PoolSize (l8utils/go/utils/workers.Workers); a
// sync.WaitGroup provides the fan-in half (wait for the whole batch to
// finish before the next tick) -- workers.Workers is bounded but
// fire-and-forget, and l8utils' other pool primitive (MultiTask) is
// fan-in but unboundedly parallel, so neither alone gives both properties
// this harness needs.
func Run(cfg Config, vnic ifs.IVNic, stop <-chan struct{}) {
	tick(cfg, vnic)
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			tick(cfg, vnic)
		}
	}
}

func tick(cfg Config, vnic ifs.IVNic) {
	items, err := l8common.GetEntitiesByQuery(cfg.ServiceName, cfg.ServiceArea, cfg.Query, vnic)
	if err != nil {
		vnic.Resources().Logger().Error("pollworker(", cfg.ServiceName, "): query failed: ", err.Error())
		return
	}
	if len(items) == 0 {
		return
	}

	pool := workers.NewWorkers(cfg.PoolSize)
	var wg sync.WaitGroup
	for _, it := range items {
		item := it
		// GetEntitiesByQuery has been observed (verified against a real
		// cluster, see scommon.PrepareImageRef) to return a one-element
		// slice holding a nil entry for a genuine zero-match query rather
		// than an empty slice -- guard against dispatching that nil to
		// Claim/Work, which don't expect it.
		if item == nil {
			continue
		}
		if cfg.Claim != nil && !cfg.Claim(item, vnic) {
			continue
		}
		wg.Add(1)
		pool.Run(&unit{fn: func() {
			defer wg.Done()
			cfg.Work(item, vnic)
		}})
	}
	wg.Wait()
}

// unit adapts a func() to workers.IWorker.
type unit struct{ fn func() }

func (u *unit) Run() { u.fn() }
