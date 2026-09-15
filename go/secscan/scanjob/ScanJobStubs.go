package scanjob

import (
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

func (this *ScanJobHandler) Put(ifs.IElements, ifs.IVNic) ifs.IElements {
	return object.NewError("not supported")
}

func (this *ScanJobHandler) Patch(ifs.IElements, ifs.IVNic) ifs.IElements {
	return object.NewError("not supported")
}

func (this *ScanJobHandler) Delete(ifs.IElements, ifs.IVNic) ifs.IElements {
	return object.NewError("not supported")
}

func (this *ScanJobHandler) Get(ifs.IElements, ifs.IVNic) ifs.IElements {
	return object.NewError("not supported")
}

func (this *ScanJobHandler) Failed(ifs.IElements, ifs.IVNic, *ifs.Message) ifs.IElements {
	return nil
}
