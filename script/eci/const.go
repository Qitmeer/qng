package main

type ContainerGroupStatus string

const (
	Scheduling     = ContainerGroupStatus("Scheduling")
	Pending        = ContainerGroupStatus("Pending")
	Running        = ContainerGroupStatus("Running")
	Succeeded      = ContainerGroupStatus("Succeeded")
	Failed         = ContainerGroupStatus("Failed")
	Restarting     = ContainerGroupStatus("Restarting")
	Updating       = ContainerGroupStatus("Updating")
	ScheduleFailed = ContainerGroupStatus("ScheduleFailed")
)

type ContainerStatus string

const (
	waiting    = ContainerStatus("Waiting")
	running    = ContainerStatus("Running")
	terminated = ContainerStatus("Terminated")
)
