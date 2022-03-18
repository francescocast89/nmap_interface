package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type LiveProgressReader struct {
	ctx                     context.Context
	mux                     sync.Mutex
	lpChannels              map[string]chan float32
	progress                map[string]float32
	progressRefreshInterval time.Duration
}

func NewLiveProgressReader(ctx context.Context, t time.Duration) *LiveProgressReader {
	obj := &LiveProgressReader{
		ctx:                     ctx,
		lpChannels:              make(map[string]chan float32, 0),
		progress:                make(map[string]float32, 0),
		progressRefreshInterval: t,
	}
	return obj
}

func (lpr *LiveProgressReader) AddJobToLiveProgressReader(j string, ch chan float32) {
	lpr.mux.Lock()
	defer lpr.mux.Unlock()
	if _, ok := lpr.lpChannels[j]; !ok {
		lpr.lpChannels[j] = ch
		go lpr.RunLiveProgress(j, ch)
	}
}

func (lpr *LiveProgressReader) AddCurrentValueToProgress(idx string, val float32) {
	lpr.mux.Lock()
	defer lpr.mux.Unlock()
	lpr.progress[idx] = val

}

func (lpr *LiveProgressReader) RemoveFromLiveProgress(idx string) {
	lpr.mux.Lock()
	defer lpr.mux.Unlock()
	delete(lpr.progress, idx)

}

func (lpr *LiveProgressReader) RunLiveProgress(idx string, c chan float32) {
	gatherTicker := time.NewTicker(lpr.progressRefreshInterval)
	defer gatherTicker.Stop()
	for {
		select {
		case <-gatherTicker.C:
			select {
			case val, ok := <-c:
				if !ok {
					lpr.RemoveFromLiveProgress(idx)
					return
				}
				lpr.AddCurrentValueToProgress(idx, val)
			}
		case <-lpr.ctx.Done():
			fmt.Println("RunLiveProgress() terminated by context")
			return
		}
	}
}

func (lpr *LiveProgressReader) PrintCurrentProgress(ctx context.Context, t time.Duration) {
	printTicker := time.NewTicker(t)
	defer printTicker.Stop()
	for {
		select {
		case <-printTicker.C:
			for idx, val := range lpr.progress {
				fmt.Println(idx, ":", val)
			}
		case <-ctx.Done():
			fmt.Println("PrintCurrentProgress() terminated by context")
			return
		}
	}
}
