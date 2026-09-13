package imageref

import (
	"errors"

	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

// PRD §9: Before() on POST calls the shared ingestion helper (§6.1 Phase
// A -- parse/dedupe/find-or-create-group/generate-ID/set-PENDING), used
// both by a direct single-entity POST and, via a plain POST retry inside
// ImgRefAdd, by the bulk handler. ID generation lives inside
// scommon.PrepareImageRef, not a separate setID step, so this callback
// uses the raw builder (not NewValidation, which would auto-generate the
// ID before PrepareImageRef's own dedupe/find-or-create logic runs).
//
// After() on PUT recomputes the parent ImageGroup rollup cache whenever
// buildDate resolves from 0 to a real value (§6.1 Phase B) or a scan
// completes (§13) -- recomputed unconditionally on every PUT since
// After() has no before/after diff to detect the specific transition
// (scommon.RecomputeImageGroupCache is idempotent).
func newImageRefServiceCallback() ifs.IServiceCallback {
	return l8common.NewServiceCallbackWithAfter(
		"ImageRef",
		func(v interface{}) bool { _, ok := v.(*secscan.ImageRef); return ok },
		func(v interface{}) {},
		nil,
		[]l8common.ActionValidateFunc{beforeImageRef},
		[]l8common.ActionValidateFunc{afterImageRef},
	)
}

func beforeImageRef(e interface{}, action ifs.Action, vnic ifs.IVNic) error {
	if action != ifs.POST {
		return nil
	}
	ref, ok := e.(*secscan.ImageRef)
	if !ok {
		return errors.New("invalid ImageRef type")
	}
	return scommon.PrepareImageRef(ref, vnic)
}

func afterImageRef(e interface{}, action ifs.Action, vnic ifs.IVNic) error {
	ref, ok := e.(*secscan.ImageRef)
	if !ok || ref == nil {
		return nil
	}
	return scommon.RecomputeImageGroupCache(ref.ImageGroupId, vnic)
}
