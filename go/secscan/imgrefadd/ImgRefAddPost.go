package imgrefadd

import (
	"strings"

	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

// Post parses the pasted image-reference lines and, per non-blank line,
// POSTs a bare ImageRef and lets ImageRefServiceCallback's Before() hook
// do the actual parsing-into-fields/dedupe/find-or-create work (§6.1 "one
// common Go helper" -- not duplicated here; this handler only decides how
// to report each line's outcome).
func (this *ImgRefAdd) Post(elems ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	req, ok := elems.Element().(*secscan.ImgRefAddRequest)
	if !ok {
		return object.NewError("invalid ImgRefAdd request type")
	}
	if req.CustomerId == "" {
		return object.NewError("customerId is required")
	}

	resp := &secscan.ImgRefAddResponse{}
	for _, raw := range req.ImageRefStrings {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		repoName, tag, digest, err := scommon.ParseImageRefString(line)
		if err != nil {
			resp.Errors = append(resp.Errors, &secscan.ImgRefAddItem{Ref: line, Reason: err.Error()})
			continue
		}

		ref := &secscan.ImageRef{
			CustomerId: req.CustomerId,
			RepoName:   repoName,
			Tag:        tag,
			Digest:     digest,
		}
		created, err := l8common.PostEntity(scommon.ImageRefServiceName, scommon.ServiceArea, ref, vnic)
		if err != nil {
			// Matched by message content, not errors.Is: the error crosses
			// the ServiceCallback -> IServiceHandler.Post -> resp.Error()
			// boundary, and this project has not verified that boundary
			// preserves error wrapping (vs. flattening to a plain string).
			if strings.Contains(err.Error(), scommon.ErrDuplicateImageRef.Error()) {
				resp.Skipped = append(resp.Skipped, &secscan.ImgRefAddItem{Ref: line, Reason: "duplicate of existing image ref"})
			} else {
				resp.Errors = append(resp.Errors, &secscan.ImgRefAddItem{Ref: line, Reason: err.Error()})
			}
			continue
		}
		createdRef, ok := created.(*secscan.ImageRef)
		if !ok || createdRef == nil {
			resp.Errors = append(resp.Errors, &secscan.ImgRefAddItem{Ref: line, Reason: "unexpected response creating ImageRef"})
			continue
		}
		resp.Created = append(resp.Created, createdRef.ImageRefId)
	}

	return object.New(nil, resp)
}
