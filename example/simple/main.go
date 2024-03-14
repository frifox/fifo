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
