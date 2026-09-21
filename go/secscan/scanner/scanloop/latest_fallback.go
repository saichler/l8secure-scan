package scanloop

import (
	"fmt"
	"strings"

	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/secscan/scanner/resolver"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

// LatestTag is the tag the MISSING fallback below looks for.
const LatestTag = "latest"

// ensureLatestRef runs when an ImageRef turns out to be MISSING, and adds
// that repo's "latest" ref when one is warranted.
//
// A MISSING image is very often a tag that no longer exists upstream while
// the repo itself is alive and still publishing -- the reported case was
// ".../istio/proxyv2:latest_1.2.0" gone from the registry while
// ".../istio/proxyv2:latest" was still there. Without this, the repo drops
// out of coverage entirely: its only ref is unscannable and nothing points
// at the tag that would work.
//
// Three steps, in this order, because each one is more expensive than the
// last:
//
//  1. If the repo already has a "latest" ref for this customer, do
//     nothing -- it is already covered, and re-adding is pointless.
//  2. Otherwise ask the registry whether "latest" actually exists. This is
//     the only network call, and it only happens for a repo that is both
//     MISSING and not already covered.
//  3. If it does, POST a bare ImageRef for it.
//
// Best-effort by design: every failure path returns quietly and leaves the
// caller to mark the original ref MISSING regardless. A registry hiccup, a
// permission problem or a losing race must not turn a MISSING image into a
// scan failure -- the fallback is a bonus, not part of the scan contract.
// It returns nothing for the same reason; there is no outcome the caller
// would act on differently.
func ensureLatestRef(missing *secscan.ImageRef, vnic ifs.IVNic) {
	// The missing ref IS the latest tag, so there is nothing to fall back
	// to -- the repo's "latest" is exactly what just failed to resolve.
	if missing.Tag == LatestTag {
		return
	}
	if missing.CustomerId == "" || missing.RepoName == "" {
		return
	}
	// These come off a stored row that passed ingestion's own check, but
	// they are about to be embedded in an L8QL string literal again and
	// this project has no verified escape mechanism for those
	// (common.ValidateNoQuote's own note), so re-check rather than assume.
	if scommon.ValidateNoQuote(missing.CustomerId, "customerId") != nil ||
		scommon.ValidateNoQuote(missing.RepoName, "repoName") != nil {
		return
	}

	covered, err := latestRefExists(missing.CustomerId, missing.RepoName, vnic)
	if err != nil {
		vnic.Resources().Logger().Error("scanloop: failed to look for an existing latest ref for ",
			missing.RepoName, ": ", err.Error())
		return
	}
	if covered {
		return
	}

	// LookupImageMeta is the project's one registry-metadata seam (it
	// already carries the pod's credentials via the go-containerregistry
	// keychain, and tests reassign it) -- a successful lookup is proof the
	// tag resolves, so this needs no second registry client of its own.
	if _, err := resolver.LookupImageMeta(missing.RepoName, LatestTag, ""); err != nil {
		// Not in the registry either. Nothing to add; the original ref
		// stays MISSING and that is the whole story.
		return
	}

	added := &secscan.ImageRef{
		CustomerId: missing.CustomerId,
		RepoName:   missing.RepoName,
		Tag:        LatestTag,
	}
	// A bare POST, not PrepareImageRef directly: ImageRefServiceCallback's
	// POST path is what calls that helper, so going through PostEntity
	// reuses the one ingestion path (dedupe, find-or-create ImageGroup,
	// PENDING/buildDate-0 defaults) instead of a second copy of it -- the
	// same reason ImgRefAdd POSTs a bare ref rather than preparing one.
	if _, err := l8common.PostEntity(scommon.ImageRefServiceName, scommon.ServiceArea, added, vnic); err != nil {
		// Losing a race against another worker (or another job) that added
		// the same ref between the check above and here is an expected
		// outcome, not a problem -- the ref exists either way, which is all
		// this function wanted. Matched on message text because the error
		// crosses the ServiceCallback -> IServiceHandler.Post -> resp.Error()
		// boundary, the same way ImgRefAdd matches it.
		if strings.Contains(err.Error(), scommon.ErrDuplicateImageRef.Error()) {
			return
		}
		vnic.Resources().Logger().Error("scanloop: failed to add ", LatestTag, " ref for ",
			missing.RepoName, ": ", err.Error())
		return
	}

	vnic.Resources().Logger().Info("scanloop: ", missing.RepoName, ":", missing.Tag,
		" is missing; added ", missing.RepoName, ":", LatestTag, " which the registry does have")
}

// latestRefExists reports whether this customer already has an ImageRef for
// repoName at the "latest" tag.
func latestRefExists(customerId, repoName string, vnic ifs.IVNic) (bool, error) {
	query := fmt.Sprintf("select * from ImageRef where customerId='%s' and repoName='%s' and tag='%s'",
		customerId, repoName, LatestTag)
	existing, err := l8common.GetEntitiesByQuery(scommon.ImageRefServiceName, scommon.ServiceArea, query, vnic)
	if err != nil {
		return false, err
	}
	// GetEntitiesByQuery's local-handler fast path returns a one-element
	// slice holding a single nil for a genuine zero-match query rather than
	// an empty slice, so len() alone is not a "found one" check -- the same
	// guard, and the same reason, as PrepareImageRef's dedupe loop.
	for _, e := range existing {
		if r, ok := e.(*secscan.ImageRef); ok && r != nil {
			return true, nil
		}
	}
	return false, nil
}
