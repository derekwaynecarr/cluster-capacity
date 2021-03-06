package apiserver

import (
	"fmt"
	"sync"

	"github.com/ingvagabund/cluster-capacity/pkg/framework"
)

var MAXWATCHERS = 10

type WatchChannelDistributor struct {
	inputChannel   chan *framework.Report
	outputChannels []chan *framework.Report
	mux            sync.Mutex
}

func NewWatchChannelDistributor(input chan *framework.Report) *WatchChannelDistributor {
	return &WatchChannelDistributor{
		inputChannel:   input,
		outputChannels: make([]chan *framework.Report, 0),
	}
}

func (w *WatchChannelDistributor) Run() {
	for {
		select {
		case report := <-w.inputChannel:
			for i := 0; i < len(w.outputChannels); i++ {
				if w.outputChannels[i] != nil {
					w.outputChannels[i] <- report
				}

			}

		}
	}
}

func (w *WatchChannelDistributor) AddChannel(ch chan *framework.Report) (int, error) {
	w.mux.Lock()
	defer w.mux.Unlock()

	for i := 0; i < len(w.outputChannels); i++ {
		if w.outputChannels[i] == nil {
			w.outputChannels[i] = ch
			return i, nil
		}
	}
	if len(w.outputChannels) >= MAXWATCHERS {
		return -1, fmt.Errorf("Maximal number of watches exceeded\n")
	}
	w.outputChannels = append(w.outputChannels, ch)
	return len(w.outputChannels) - 1, nil
}

func (w *WatchChannelDistributor) RemoveChannel(pos int) {
	w.mux.Lock()
	defer w.mux.Unlock()
	close(w.outputChannels[pos])
	w.outputChannels[pos] = nil
}
