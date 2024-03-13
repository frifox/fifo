package fifo

import (
	"container/list"
	"context"
	"sync"
)

type Queue[ID comparable, Request any, Response any] struct {
	Jobs chan job[ID, Request]

	jobs     map[ID]job[ID, Request]
	jobsLock sync.Mutex

	closures     map[ID][]func(Response)
	closuresLock sync.Mutex

	push chan<- ID
	pull <-chan ID

	ctx  context.Context
	done context.CancelFunc
}

type job[JobID comparable, Request any] struct {
	ID      JobID
	Request Request
}

func NewQueue[ID comparable, Request any, Response any](ctx context.Context) *Queue[ID, Request, Response] {
	queue := &Queue[ID, Request, Response]{}
	queue.init(ctx)
	return queue
}

func (q *Queue[ID, Request, Response]) init(ctx context.Context) {
	q.jobs = map[ID]job[ID, Request]{}
	q.closures = map[ID][]func(Response){}

	q.push, q.pull = q.newQueue()

	q.ctx, q.done = context.WithCancel(ctx)

	q.Jobs = make(chan job[ID, Request])
	go q.pushJobsToConsumers()
}

func (q *Queue[ID, Request, Response]) Add(jobID ID, request Request, closures ...func(Response)) (queued bool) {
	q.jobsLock.Lock()
	defer q.jobsLock.Unlock()

	// register closure, if any
	if len(closures) > 0 {
		q.closuresLock.Lock()
		q.closures[jobID] = append(q.closures[jobID], closures...)
		q.closuresLock.Unlock()
	}

	// if key already queued, don't re-queue
	_, ok := q.jobs[jobID]
	if ok {
		return false
	}
	// otherwise, add item to jobs
	q.jobs[jobID] = job[ID, Request]{
		ID:      jobID,
		Request: request,
	}
	q.push <- jobID

	return true
}

func (q *Queue[ID, Request, Response]) AddAndCloseOnce(jobID ID, request Request, closures ...func(Response)) (queued bool) {
	q.jobsLock.Lock()
	defer q.jobsLock.Unlock()

	// if key already queued, don't re-queue and don't register closures
	_, ok := q.jobs[jobID]
	if ok {
		return false
	}

	// register closure, if any
	if len(closures) > 0 {
		q.closuresLock.Lock()
		q.closures[jobID] = append(q.closures[jobID], closures...)
		q.closuresLock.Unlock()
	}

	// add item to jobs
	q.jobs[jobID] = job[ID, Request]{
		ID:      jobID,
		Request: request,
	}
	q.push <- jobID

	return true
}

func (q *Queue[ID, Request, Response]) get(key ID) (job job[ID, Request]) {
	q.jobsLock.Lock()
	job = q.jobs[key]
	q.jobsLock.Unlock()

	return job
}
func (q *Queue[ID, Request, Response]) Finish(jobID ID, ret Response) {
	// remove item from jobs
	q.jobsLock.Lock()
	delete(q.jobs, jobID)
	defer q.jobsLock.Unlock()

	// execute closures if exist
	q.closuresLock.Lock()
	for _, closure := range q.closures[jobID] {
		go closure(ret)
	}
	delete(q.closures, jobID)
	q.closuresLock.Unlock()
}

func (q *Queue[ID, Request, Response]) pushJobsToConsumers() {
	for {
		select {
		case <-q.ctx.Done():
			close(q.push) // will close q.manageQueue()
			close(q.Jobs) // will close q.Requests consumers
			return
		case key := <-q.pull: // pull K from queue
			q.Jobs <- q.get(key) // and push related V to consumer
		}
	}
}

// newQueue based on https://github.com/Symantec/Dominator/blob/master/lib/queue/dataQueue.go
func (q *Queue[ID, Request, Response]) newQueue() (chan<- ID, <-chan ID) {
	push := make(chan ID, 1)
	pull := make(chan ID, 1)
	go q.manageQueue(push, pull)
	return push, pull
}

// manageQueue, clever use of push/pull chan in front, container/list in back
func (q *Queue[ID, Request, Response]) manageQueue(push <-chan ID, pull chan<- ID) {
	queue := list.New()
	for {
		front := queue.Front() // does queue have anything in it?
		if front != nil {
			select {
			case pull <- front.Value.(ID): // consumer is pulling, give it K
				queue.Remove(front)
			case value, ok := <-push: // adding new K to queue
				if ok {
					queue.PushBack(value)
				} else {
					push = nil // push chan is closed, we'll close pull chan when queue is empty
				}
			}
		} else {
			if push == nil {
				close(pull) // queue is empty & push chan is closed, we're done
				return
			}
			value, ok := <-push // wait for new K to add to queue
			if !ok {
				close(pull) // push chan closed while waiting. close pull chan, we're done
				return
			}
			queue.PushBack(value) // add this new K to queue
		}
	}
}
