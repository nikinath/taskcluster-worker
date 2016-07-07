// This source code file is AUTO-GENERATED by github.com/taskcluster/jsonschema2go

package schedulerevents

type (
	// Message that all reruns of a task has failed it is now blocking the task-graph from finishing.
	//
	// See http://schemas.taskcluster.net/scheduler/v1/task-graph-blocked-message.json#
	BlockedTaskGraphMessage struct {

		// See http://schemas.taskcluster.net/scheduler/v1/task-graph-blocked-message.json#/properties/status
		Status TaskGraphStatusStructure `json:"status"`

		// Unique `taskId` that is blocking this task-graph from completion.
		//
		// Syntax:     ^[A-Za-z0-9_-]{8}[Q-T][A-Za-z0-9_-][CGKOSWaeimquy26-][A-Za-z0-9_-]{10}[AQgw]$
		//
		// See http://schemas.taskcluster.net/scheduler/v1/task-graph-blocked-message.json#/properties/taskId
		TaskID string `json:"taskId"`

		// Message version
		//
		// Possible values:
		//   * 1
		//
		// See http://schemas.taskcluster.net/scheduler/v1/task-graph-blocked-message.json#/properties/version
		Version int `json:"version"`
	}

	// Messages as posted to `scheduler/v1/task-graph-running` informing the world that a new task-graph have been submitted.
	//
	// See http://schemas.taskcluster.net/scheduler/v1/task-graph-running-message.json#
	NewTaskGraphMessage struct {

		// See http://schemas.taskcluster.net/scheduler/v1/task-graph-running-message.json#/properties/status
		Status TaskGraphStatusStructure `json:"status"`

		// Message version
		//
		// Possible values:
		//   * 1
		//
		// See http://schemas.taskcluster.net/scheduler/v1/task-graph-running-message.json#/properties/version
		Version int `json:"version"`
	}

	// Messages as posted to `scheduler/v1/task-graph-extended` informing the world that a task-graph have been extended.
	//
	// See http://schemas.taskcluster.net/scheduler/v1/task-graph-extended-message.json#
	TaskGraphExtendedMessage struct {

		// See http://schemas.taskcluster.net/scheduler/v1/task-graph-extended-message.json#/properties/status
		Status TaskGraphStatusStructure `json:"status"`

		// Message version
		//
		// Possible values:
		//   * 1
		//
		// See http://schemas.taskcluster.net/scheduler/v1/task-graph-extended-message.json#/properties/version
		Version int `json:"version"`
	}

	// Message that all tasks in a task-graph have now completed successfully and the graph is _finished_.
	//
	// See http://schemas.taskcluster.net/scheduler/v1/task-graph-finished-message.json#
	TaskGraphFinishedMessage struct {

		// See http://schemas.taskcluster.net/scheduler/v1/task-graph-finished-message.json#/properties/status
		Status TaskGraphStatusStructure `json:"status"`

		// Message version
		//
		// Possible values:
		//   * 1
		//
		// See http://schemas.taskcluster.net/scheduler/v1/task-graph-finished-message.json#/properties/version
		Version int `json:"version"`
	}

	// A representation of **task-graph status** as known by the scheduler, without the state of all individual tasks.
	//
	// See http://schemas.taskcluster.net/scheduler/v1/task-graph-status.json#
	TaskGraphStatusStructure struct {

		// Unique identifier for task-graph scheduler managing the given task-graph
		//
		// Syntax:     ^([a-zA-Z0-9-_]*)$
		// Min length: 1
		// Max length: 22
		//
		// See http://schemas.taskcluster.net/scheduler/v1/task-graph-status.json#/properties/schedulerId
		SchedulerID string `json:"schedulerId"`

		// Task-graph state, this enum is **frozen** new values will **not** be added.
		//
		// Possible values:
		//   * "running"
		//   * "blocked"
		//   * "finished"
		//
		// See http://schemas.taskcluster.net/scheduler/v1/task-graph-status.json#/properties/state
		State string `json:"state"`

		// Unique task-graph identifier, this is UUID encoded as [URL-safe base64](http://tools.ietf.org/html/rfc4648#section-5) and stripped of `=` padding.
		//
		// Syntax:     ^[A-Za-z0-9_-]{8}[Q-T][A-Za-z0-9_-][CGKOSWaeimquy26-][A-Za-z0-9_-]{10}[AQgw]$
		//
		// See http://schemas.taskcluster.net/scheduler/v1/task-graph-status.json#/properties/taskGraphId
		TaskGraphID string `json:"taskGraphId"`
	}
)