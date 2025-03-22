package main

// runJob is called by the cluster scheduler defined in plugin.go.
// Although this appears unused, it's referenced through a function pointer
// in the cluster.Schedule call.
//
//nolint:unused
func (p *Plugin) runJob() {
	// Include job logic here
	p.API.LogInfo("Job is currently running")
}
