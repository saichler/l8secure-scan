package vulnrep

import (
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

func (this *VulnRep) Put(ifs.IElements, ifs.IVNic) ifs.IElements {
	return object.NewError("not supported")
}

func (this *VulnRep) Patch(ifs.IElements, ifs.IVNic) ifs.IElements {
	return object.NewError("not supported")
}

func (this *VulnRep) Delete(ifs.IElements, ifs.IVNic) ifs.IElements {
	return object.NewError("not supported")
}

func (this *VulnRep) Get(ifs.IElements, ifs.IVNic) ifs.IElements {
	return object.NewError("not supported")
}

func (this *VulnRep) Failed(ifs.IElements, ifs.IVNic, *ifs.Message) ifs.IElements {
	return nil
}
