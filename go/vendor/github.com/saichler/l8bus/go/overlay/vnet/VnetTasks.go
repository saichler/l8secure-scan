package vnet

import (
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/queues"
)

type VnetTask struct {
	vnic ifs.IVNic
	data []byte
}

type QueueDest int

const (
	QSystem       QueueDest = 1
	QService      QueueDest = 2
	QHandleData   QueueDest = 3
	QHealthReport QueueDest = 4
)

func (this *VNet) addVnetTask(dest QueueDest, data []byte, vnic ifs.IVNic) {
	if vnic == nil {
		return
	}
	switch dest {
	case QSystem:
		this.vnetSystemTasks.Add(&VnetTask{vnic: vnic, data: data})
	case QService:
		this.vnetServiceTasks.Add(&VnetTask{vnic: vnic, data: data})
	case QHandleData:
		this.handleDataTasks.Add(&VnetTask{vnic: vnic, data: data})
	case QHealthReport:
		this.healthReport.Add(&VnetTask{vnic: vnic, data: data})
	}
}

func (this *VNet) processTasks(queue *queues.Queue, f func(data []byte, vnic ifs.IVNic)) {
	for this.running {
		tsk := queue.Next()
		if tsk != nil {
			task := tsk.(*VnetTask)
			f(task.data, task.vnic)
		}
	}
}
