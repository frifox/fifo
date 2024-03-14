package main

import (
	"context"
	"errors"
	"github.com/frifox/fifo"
	"net/http"
	"strings"
	"time"
)

type Worker struct {
	AppCtx context.Context
	Queue  *fifo.Queue[string, any, any]

	client             http.Client
	fastRequestTimeout time.Duration
	slowRequestTimeout time.Duration
}

type WorkerRequestA struct {
	Email string
}
type WorkerRequestB struct {
	Email string
}

func (c *Worker) Run() {
	for job := range c.Queue.Jobs {
		switch request := job.Request.(type) {
		case WorkerRequestA:
			ctx, cancel := context.WithTimeout(AppCtx, c.slowRequestTimeout)
			response := c.processRequestA(request, ctx)
			cancel()
			c.Queue.Finish(job.ID, response)
		case WorkerRequestB:
			ctx, cancel := context.WithTimeout(AppCtx, c.fastRequestTimeout)
			response := c.processRequestB(request, ctx)
			cancel()
			c.Queue.Finish(job.ID, response)
		default:
			c.Queue.Finish(job.ID, errors.New("unsupported request type"))
		}
	}
}

type WorkerResponse struct {
	IsGmail bool
}

func (c *Worker) processRequestA(request WorkerRequestA, ctx context.Context) any {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Second * 1): // simulate SLOW func
		return WorkerResponse{
			IsGmail: strings.HasSuffix(request.Email, "@gmail.com"),
		}
	}
}

func (c *Worker) processRequestB(request WorkerRequestB, ctx context.Context) any {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Millisecond * 100): // simulate FAST func
		return WorkerResponse{
			IsGmail: strings.HasSuffix(request.Email, "@gmail.com"),
		}
	}
}
