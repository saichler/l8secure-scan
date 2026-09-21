// Package resolver is the metadata resolver loop (PRD §6.1 Phase B): fills
// in ImageRef.buildDate for newly-added refs by querying the registry for
// the image's Created timestamp.
package resolver

import (
	"errors"
	"time"

	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/secscan/scanner/pollworker"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

const (
	pollInterval = 15 * time.Second
	poolSize     = 4
)

// Run starts the resolver loop and blocks until stop is closed.
func Run(vnic ifs.IVNic, stop <-chan struct{}) {
	pollworker.Run(pollworker.Config{
		ServiceName: scommon.ImageRefServiceName,
		ServiceArea: scommon.ServiceArea,
		// sizeBytes, not buildDate. A new ref has both at 0, so it is
		// picked up either way -- but a ref resolved before size_bytes
		// existed has a buildDate and a zero size, and querying buildDate
		// would never look at it again. L8QL supports only AND-chained
		// equality (no OR), so one predicate has to cover both, and this
		// is the one that does. A real image cannot have a zero
		// compressed size, so nothing loops on a successful resolve.
		Query:    "select * from ImageRef where sizeBytes=0",
		Interval: pollInterval,
		PoolSize: poolSize,
		Claim:    nil, // idempotent, safe to retry -- no exclusive claim needed
		Work:     work,
	}, vnic, stop)
}

func work(item interface{}, vnic ifs.IVNic) {
	ref, ok := item.(*secscan.ImageRef)
	if !ok || ref == nil {
		return
	}

	meta, err := LookupImageMeta(ref.RepoName, ref.Tag, ref.Digest)
	if err != nil {
		ref.ScanError = err.Error()
	} else {
		ref.BuildDate = meta.Created
		ref.SizeBytes = meta.SizeBytes
		ref.ScanError = ""
	}

	if putErr := l8common.PutEntity(scommon.ImageRefServiceName, scommon.ServiceArea, ref, vnic); putErr != nil {
		vnic.Resources().Logger().Error("resolver: failed to update ImageRef ", ref.ImageRefId, ": ", putErr.Error())
	}
}

// ImageMeta is what one registry lookup yields about an image. Both
// fields come from the SAME remote.Image call -- the manifest is already
// fetched to read the config, so the size costs no extra round trip.
type ImageMeta struct {
	// Created is the config's Created timestamp, as a Unix second.
	Created int64
	// SizeBytes is the compressed size the registry reports: the config
	// blob plus every layer blob. Not the extracted-on-disk size, which a
	// registry cannot report without the image being pulled.
	SizeBytes int64
}

// LookupImageMeta queries the registry (credentials already present on the
// pod, PRD §13/§18) for an image's metadata, pure-Go via
// go-containerregistry so the scanner pod needs no Docker daemon.
// Exported as a reassignable var (default: the real implementation below)
// so PRD §19's metadata-resolution tests can inject a fixture registry
// client without calling any unexported function
// (TestLocationAndApproach) -- tests only ever drive this through the
// exported resolver.Run entry point.
var LookupImageMeta = lookupImageMetaRemote

func lookupImageMetaRemote(repoName, tag, digest string) (*ImageMeta, error) {
	refStr := repoName
	switch {
	case tag != "":
		refStr = repoName + ":" + tag
	case digest != "":
		refStr = repoName + "@" + digest
	default:
		return nil, errors.New("image reference has neither tag nor digest")
	}

	ref, err := name.ParseReference(refStr)
	if err != nil {
		return nil, err
	}
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, err
	}
	cfg, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}
	manifest, err := img.Manifest()
	if err != nil {
		return nil, err
	}
	// Config blob + every layer blob, which is what a registry means by an
	// image's size. img.Size() is NOT this -- that is the size of the
	// manifest document itself, a few hundred bytes.
	size := manifest.Config.Size
	for _, layer := range manifest.Layers {
		size += layer.Size
	}
	return &ImageMeta{Created: cfg.Created.Time.Unix(), SizeBytes: size}, nil
}
