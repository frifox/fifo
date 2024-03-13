package fifo

import (
	"context"
	"sync"
)

type Queue[ID comparable, Request any, Response any] struct {
	Jobs chan queueJob[ID, Request]

	jobs     map[ID]queueJob[ID, Request]
	jobsLock sync.Mutex

	closures     map[ID][]func(Response)
	closuresLock sync.Mutex

	push chan<- ID
	pull <-chan ID

	ctx  context.Context
	done context.CancelFunc
}

type queueJob[JobID comparable, Request any] struct {
	ID      JobID
	Request Request
}

func NewQueue[ID comparable, Request any, Response any](ctx context.Context) *Queue[ID, Request, Response] {
	queue := &Queue[ID, Request, Response]{}
	queue.init(ctx)
	return queue
}
