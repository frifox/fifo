# FIFO Queue with dedup & closures

When you get too many requests and really need to avoid processing same job multiple times.

## Init queue
Queue for jobs of type `Request`, results of type `Response`, and jobs uniquely grouped by `JobID`\
`queue := fifo.NewQueue[JobID,Request,Response](context.Background())`

## Send job to queue
queue a job for background processing:\
`queue.Add(jobID, myRequest)`

queue the job and execute all queued `someClosureFunc` once finished:\
`queue.Add(jobID, myRequest, someClosureFunc)`

queue the job and execute only the first `someClosureFunc` once finished:\
`queue.AddAndCloseOnce(jobID, myRquest, someClosureFunc)`

## PS
Use `fifo.Queue[comparable, any, any](ctx)` if you need more than one `Requset`/`Response` type and do type assertion at runtime:\
[example/advanced/main.go](https://github.com/frifox/fifo/blob/master/example/advanced/main.go#L61)

## Example (simple)
```go
package main

import (
	"context"
	"fmt"
	"github.com/frifox/fifo"
	"io"
	"net/http"
	"sync"
)

type Request struct {
	URL string
}
type Response struct {
	Body string
}

func main() {
	queue := fifo.NewQueue[string, Request, Response](context.Background())
	tasks := sync.WaitGroup{}

	request := Request{
		URL: "https://google.com/",
	}
	closure := func(r Response) {
		fmt.Printf("[closure] job finished: body=`%s`\n", r.Body)
		tasks.Done()
	}

	// launch 10 queue workers
	for i := 0; i < 10; i++ {
		go func() {
			for job := range queue.Jobs {
				fmt.Printf("[worker] starting: jobID=%s\n", job.ID)
				resp, _ := http.Get(job.Request.URL)
				body, _ := io.ReadAll(resp.Body)

				fmt.Printf("[worker] processed: jobID=%s\n", job.ID)
				queue.Finish(job.ID, Response{
					Body: string(body[:20]) + "...",
				})
			}
		}()
	}

	// queue job 10 times
	for i := 0; i < 10; i++ {
		tasks.Add(1)
		queued := queue.Add(request.URL, request, closure)
		if queued {
			fmt.Printf("[queue] job added\n")
		} else {
			fmt.Printf("[queue] job is already in the queue\n")
		}
	}

	tasks.Wait()
}
```

```console
user@pc:~$ go run example/simple/main.go
[queue] job added
[queue] job is already in the queue
[queue] job is already in the queue
[queue] job is already in the queue
[queue] job is already in the queue
[queue] job is already in the queue
[queue] job is already in the queue
[queue] job is already in the queue
[queue] job is already in the queue
[queue] job is already in the queue
[worker] starting: jobID=https://google.com/
[worker] processed: jobID=https://google.com/
[closure] job finished: body=`<!doctype html><html...`
[closure] job finished: body=`<!doctype html><html...`
[closure] job finished: body=`<!doctype html><html...`
[closure] job finished: body=`<!doctype html><html...`
[closure] job finished: body=`<!doctype html><html...`
[closure] job finished: body=`<!doctype html><html...`
[closure] job finished: body=`<!doctype html><html...`
[closure] job finished: body=`<!doctype html><html...`
[closure] job finished: body=`<!doctype html><html...`
[closure] job finished: body=`<!doctype html><html...`

```