package agent

import (
	"sync"
)

type WaitGroupWrapper struct {
	sync.WaitGroup
}

//简单的对原生的waitgroup进行封装
func (w *WaitGroupWrapper) Wrap(sf func()) {
	w.Add(1)
	go func() {
		sf()
		w.Done()
	}()
}
