package imgrefadd

import (
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

func (this *ImgRefAdd) Put(ifs.IElements, ifs.IVNic) ifs.IElements {
	return object.NewError("not supported")
}

func (this *ImgRefAdd) Patch(ifs.IElements, ifs.IVNic) ifs.IElements {
	return object.NewError("not supported")
}

func (this *ImgRefAdd) Delete(ifs.IElements, ifs.IVNic) ifs.IElements {
	return object.NewError("not supported")
}

func (this *ImgRefAdd) Get(ifs.IElements, ifs.IVNic) ifs.IElements {
	return object.NewError("not supported")
}

func (this *ImgRefAdd) Failed(ifs.IElements, ifs.IVNic, *ifs.Message) ifs.IElements {
	return nil
}
