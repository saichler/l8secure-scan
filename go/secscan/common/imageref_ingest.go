package common

import (
	"errors"
	"fmt"
	"strings"

	l8common "github.com/saichler/l8common/go/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

// ErrDuplicateImageRef is returned by PrepareImageRef when an ImageRef
// already exists for (customerId, repoName, tag, digest) -- PRD §6.1
// step 3. ImgRefAdd (go/secscan/imgrefadd) matches on this error's message
// text rather than errors.Is, since the error crosses the ServiceCallback
// -> IServiceHandler.Post -> resp.Error() boundary and that boundary's
// error-wrapping behavior is unverified.
var ErrDuplicateImageRef = errors.New("duplicate of existing image ref")

// ValidateNoQuote rejects a value that would break out of an L8QL string
// literal (single-quote delimited) if embedded directly in a query -- this
// project has no verified escape mechanism for L8QL string literals, so
// user-supplied values that would flow into a WHERE clause (customerId,
// repoName, tag, digest) are rejected outright rather than "escaped".
func ValidateNoQuote(value, fieldName string) error {
	if strings.ContainsRune(value, '\'') {
		return errors.New(fieldName + " must not contain a single-quote character")
	}
	return nil
}

// ParseImageRefString parses one pasted image reference (PRD §5) into its
// repoName/tag/digest parts, e.g.
// "registry.example.com/myorg/backend:v1.4.2@sha256:abc" ->
// repoName="registry.example.com/myorg/backend", tag="v1.4.2", digest="sha256:abc".
// A tag separator ':' only counts inside the last path segment, so a
// registry host:port (e.g. "localhost:5000/myimage:latest") is not
// mistaken for a tag.
func ParseImageRefString(raw string) (repoName, tag, digest string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", errors.New("empty image reference")
	}
	if err = ValidateNoQuote(raw, "image reference"); err != nil {
		return "", "", "", err
	}

	left := raw
	if idx := strings.Index(raw, "@"); idx >= 0 {
		left = raw[:idx]
		digest = raw[idx+1:]
		if digest == "" {
			return "", "", "", errors.New("empty digest after '@'")
		}
	}
	if left == "" {
		return "", "", "", errors.New("could not parse image reference")
	}

	repoName = left
	tagRegion := left
	tagRegionOffset := 0
	if slash := strings.LastIndex(left, "/"); slash >= 0 {
		tagRegion = left[slash+1:]
		tagRegionOffset = slash + 1
	}
	if colon := strings.LastIndex(tagRegion, ":"); colon >= 0 {
		repoName = left[:tagRegionOffset+colon]
		tag = left[tagRegionOffset+colon+1:]
		if tag == "" {
			return "", "", "", errors.New("empty tag after ':'")
		}
	}
	if repoName == "" {
		return "", "", "", errors.New("could not parse repo name")
	}
	return repoName, tag, digest, nil
}

// deriveImageName is the grouping key (PRD §5): the last path segment of
// repoName, lower-cased.
func deriveImageName(repoName string) string {
	name := repoName
	if idx := strings.LastIndex(repoName, "/"); idx >= 0 {
		name = repoName[idx+1:]
	}
	return strings.ToLower(name)
}

// PrepareImageRef is the single shared "Phase A" ingestion helper (PRD
// §6.1) run for every new ImageRef, regardless of entry point: it derives
// imageName, dedupes against existing rows, finds-or-creates the parent
// ImageGroup, and sets the PENDING/unresolved-build-date defaults. The
// ONLY caller is ImageRefServiceCallback's POST path -- ImgRefAdd does not
// call this directly; it POSTs a bare ImageRef and relies on that same
// callback path, per "one common Go helper" (Duplication Prevention,
// Second Instance Rule): a second copy of this logic in ImgRefAdd would
// re-run the dedupe query against not-yet-committed state for no benefit.
func PrepareImageRef(ref *secscan.ImageRef, vnic ifs.IVNic) error {
	if ref.CustomerId == "" {
		return errors.New("customerId is required")
	}
	if ref.RepoName == "" {
		return errors.New("repoName is required")
	}
	for _, f := range []struct{ v, name string }{
		{ref.CustomerId, "customerId"}, {ref.RepoName, "repoName"},
		{ref.Tag, "tag"}, {ref.Digest, "digest"},
	} {
		if err := ValidateNoQuote(f.v, f.name); err != nil {
			return err
		}
	}

	imageName := deriveImageName(ref.RepoName)

	dupQuery := fmt.Sprintf(
		"select * from ImageRef where customerId='%s' and repoName='%s' and tag='%s' and digest='%s'",
		ref.CustomerId, ref.RepoName, ref.Tag, ref.Digest)
	existing, err := l8common.GetEntitiesByQuery(ImageRefServiceName, ServiceArea, dupQuery, vnic)
	vnic.Resources().Logger().Info("DEBUG PrepareImageRef dupQuery=", dupQuery, " len(existing)=", len(existing), " err=", err)
	for i, e := range existing {
		vnic.Resources().Logger().Info("DEBUG PrepareImageRef existing[", i, "]=", fmt.Sprintf("%#v", e))
	}
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return ErrDuplicateImageRef
	}

	groupId, err := findOrCreateImageGroup(ref.CustomerId, imageName, vnic)
	if err != nil {
		return err
	}

	l8common.GenerateID(&ref.ImageRefId)
	ref.ImageGroupId = groupId
	ref.ScanStatus = secscan.ScanStatus_SCAN_STATUS_PENDING
	ref.BuildDate = 0
	return nil
}

func findOrCreateImageGroup(customerId, imageName string, vnic ifs.IVNic) (string, error) {
	query := fmt.Sprintf("select * from ImageGroup where customerId='%s' and imageName='%s'", customerId, imageName)
	existing, err := l8common.GetEntitiesByQuery(ImageGroupServiceName, ServiceArea, query, vnic)
	if err != nil {
		return "", err
	}
	for _, e := range existing {
		if g, ok := e.(*secscan.ImageGroup); ok && g != nil {
			return g.ImageGroupId, nil
		}
	}

	group := &secscan.ImageGroup{CustomerId: customerId, ImageName: imageName}
	created, err := l8common.PostEntity(ImageGroupServiceName, ServiceArea, group, vnic)
	if err != nil {
		return "", err
	}
	g, ok := created.(*secscan.ImageGroup)
	if !ok || g == nil {
		return "", errors.New("unexpected type returned creating ImageGroup")
	}
	return g.ImageGroupId, nil
}

// RecomputeImageGroupCache is the single hook (PRD §7/§9) that maintains
// ImageGroup's denormalized cache fields (imageRefCount, latestBuildDate,
// newestCounts, oldestCounts). Every other consumer (CSV report, dashboard,
// Group Detail Trend panel) only ever reads these cached fields -- this is
// the one place "which ImageRef is newest/oldest scanned" is computed.
// Called from ImageRefServiceCallback.After() on every PUT; recomputing
// unconditionally (rather than only on the specific buildDate-resolves or
// scan-completes transitions) is deliberate -- After() only receives the
// post-write entity, not a before/after diff, so there is no cheaper way
// to detect exactly which transition occurred, and recomputation is
// idempotent.
func RecomputeImageGroupCache(groupId string, vnic ifs.IVNic) error {
	if groupId == "" {
		return nil
	}
	query := fmt.Sprintf("select * from ImageRef where imageGroupId='%s'", groupId)
	all, err := l8common.GetEntitiesByQuery(ImageRefServiceName, ServiceArea, query, vnic)
	if err != nil {
		return err
	}

	result, err := l8common.GetEntity(ImageGroupServiceName, ServiceArea, &secscan.ImageGroup{ImageGroupId: groupId}, vnic)
	if err != nil {
		return err
	}
	g, ok := result.(*secscan.ImageGroup)
	if !ok || g == nil {
		return errors.New("image group not found: " + groupId)
	}

	var latestBuildDate int64
	var scannedRefCount int32
	var newestRef, oldestRef *secscan.ImageRef
	for _, e := range all {
		ref, ok := e.(*secscan.ImageRef)
		if !ok || ref == nil {
			continue
		}
		if ref.BuildDate > latestBuildDate {
			latestBuildDate = ref.BuildDate
		}
		if ref.ScanStatus != secscan.ScanStatus_SCAN_STATUS_COMPLETED {
			continue
		}
		scannedRefCount++
		if newestRef == nil || ref.BuildDate > newestRef.BuildDate {
			newestRef = ref
		}
		if oldestRef == nil || ref.BuildDate < oldestRef.BuildDate {
			oldestRef = ref
		}
	}

	g.ImageRefCount = int32(len(all))
	g.LatestBuildDate = latestBuildDate
	g.ScannedRefCount = scannedRefCount
	if newestRef != nil {
		g.NewestCounts = newestRef.TotalCounts
	} else {
		g.NewestCounts = nil
	}
	if oldestRef != nil {
		g.OldestCounts = oldestRef.TotalCounts
	} else {
		g.OldestCounts = nil
	}

	return l8common.PutEntity(ImageGroupServiceName, ServiceArea, g, vnic)
}
