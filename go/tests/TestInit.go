package tests

import (
	. "github.com/saichler/l8test/go/infra/t_resources"
	. "github.com/saichler/l8test/go/infra/t_topology"
	. "github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/logger"
)

// Shared test topology (TestLocationAndApproach: every test in this
// package exercises the system through IVNic/HTTP end-to-end, never by
// calling unexported functions). Mirrors ../l8alarms/go/tests/TestInit.go's
// real, working setup exactly.
var topo *TestTopology
var FLog = logger.NewLoggerDirectImpl(logger.NewFileLogMethod("test.log"))

func init() {
	Log.SetLogLevel(Trace_Level)
}

func setup() {
	setupTopology()
}

func tear() {
	shutdownTopology()
}

func setupTopology() {
	topo = NewTestTopology(4, []int{20000, 30000, 40000}, Info_Level)
}

func shutdownTopology() {
	topo.Shutdown()
}
