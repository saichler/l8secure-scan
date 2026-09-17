package imgrefdelete

import (
	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

// Post fetches the ImageRef (to learn its parent group before it's gone),
// deletes it, then recomputes the group's rollup cache explicitly --
// l8common.GetEntity/PostEntity/PutEntity all exist, but no DeleteEntity
// helper does, so the delete itself replicates GetEntity's own
// local-handler-first-then-remote-request pattern by hand.
func (this *ImgRefDelete) Post(elems ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	req, ok := elems.Element().(*secscan.ImgRefDeleteRequest)
	if !ok {
		return object.NewError("invalid ImgRefDelete request type")
	}
	if req.ImageRefId == "" {
		return object.NewError("imageRefId is required")
	}

	result, err := l8common.GetEntity(scommon.ImageRefServiceName, scommon.ServiceArea, &secscan.ImageRef{ImageRefId: req.ImageRefId}, vnic)
	if err != nil {
		return object.NewError("fetch error: " + err.Error())
	}
	ref, ok := result.(*secscan.ImageRef)
	if !ok || ref == nil {
		return object.NewError("image ref not found: " + req.ImageRefId)
	}
	groupId := ref.ImageGroupId

	if err := deleteImageRef(ref, vnic); err != nil {
		return object.NewError("delete error: " + err.Error())
	}

	if err := scommon.RecomputeImageGroupCache(groupId, vnic); err != nil {
		vnic.Resources().Logger().Error("ImgRefDelete: failed to recompute group cache ", groupId, ": ", err.Error())
	}

	return object.New(nil, &secscan.ImgRefDeleteResponse{ImageGroupId: groupId})
}

// deleteImageRef mirrors l8common.GetEntity's own local-handler-first,
// remote-request-fallback shape (same package, same ifs.IVNic-based
// service lookup) -- there is no l8common.DeleteEntity to call instead.
func deleteImageRef(ref *secscan.ImageRef, vnic ifs.IVNic) error {
	handler, ok := l8common.ServiceHandler(scommon.ImageRefServiceName, scommon.ServiceArea, vnic)
	if ok {
		resp := handler.Delete(object.New(nil, ref), vnic)
		return resp.Error()
	}
	resp := vnic.Request("", scommon.ImageRefServiceName, scommon.ServiceArea, ifs.DELETE, ref, 30)
	return resp.Error()
}
