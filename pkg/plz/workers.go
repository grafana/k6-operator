package plz

import (
	"fmt"
	"sync"
)

// PLZWorkers is a gathering point for multiple PLZ workers.
// Note: while PLZWorkers is thread-safe, a lone PLZWorker is not.
type PLZWorkers struct {
	m sync.Map
}

func (w *PLZWorkers) AddWorker(name string, worker *PLZWorker) error {
	_, loaded := w.m.LoadOrStore(name, worker)
	if loaded {
		return fmt.Errorf("PLZ worker has already been added")
	}
	return nil
}

func (w *PLZWorkers) GetWorker(name string) (worker *PLZWorker, err error) {
	ptr, ok := w.m.Load(name)
	if !ok {
		return nil, fmt.Errorf("PLZ worker doesn't exist anymore")
	}

	if worker, ok = ptr.(*PLZWorker); !ok {
		return nil, fmt.Errorf("cannot load PLZ worker: this might be a bug")
	}

	return worker, nil
}

func (w *PLZWorkers) DeleteWorker(name string) {
	w.m.Delete(name)
}
