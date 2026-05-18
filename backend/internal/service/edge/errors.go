package edge

import "errors"

var (
	ErrNoAvailableNodes   = errors.New("no available edge nodes")
	ErrNodeNotFound       = errors.New("edge node not found")
	ErrNodeAlreadyExists  = errors.New("edge node already exists")
	ErrSyncFailed         = errors.New("sync operation failed")
	ErrInvalidNodeStatus  = errors.New("invalid node status")
	ErrSchedulerRunning   = errors.New("scheduler is already running")
)