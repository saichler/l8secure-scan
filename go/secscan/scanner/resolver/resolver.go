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
		Query:       "select * from ImageRef where buildDate=0",
		Interval:    pollInterval,
		PoolSize:    poolSize,
		Claim:       nil, // idempotent, safe to retry -- no exclusive claim needed
		Work:        work,
	}, vnic, stop)
}

func work(item interface{}, vnic ifs.IVNic) {
	ref, ok := item.(*secscan.ImageRef)
	if !ok || ref == nil {
		return
	}

	created, err := lookupCreated(ref.RepoName, ref.Tag, ref.Digest)
	if err != nil {
		ref.ScanError = err.Error()
	} else {
		ref.BuildDate = created
		ref.ScanError = ""
	}

	if putErr := l8common.PutEntity(scommon.ImageRefServiceName, scommon.ServiceArea, ref, vnic); putErr != nil {
		vnic.Resources().Logger().Error("resolver: failed to update ImageRef ", ref.ImageRefId, ": ", putErr.Error())
	}
}

// lookupCreated queries the registry (credentials already present on the
// pod, PRD §13/§18) for the image's config Created timestamp, pure-Go via
// go-containerregistry so the scanner pod needs no Docker daemon.
func lookupCreated(repoName, tag, digest string) (int64, error) {
	refStr := repoName
	switch {
	case tag != "":
		refStr = repoName + ":" + tag
	case digest != "":
		refStr = repoName + "@" + digest
	default:
		return 0, errors.New("image reference has neither tag nor digest")
	}

	ref, err := name.ParseReference(refStr)
	if err != nil {
		return 0, err
	}
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return 0, err
	}
	cfg, err := img.ConfigFile()
	if err != nil {
		return 0, err
	}
	return cfg.Created.Time.Unix(), nil
}
