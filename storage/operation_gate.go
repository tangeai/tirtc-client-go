package storage

import "sync"

// nativeOperationGate serializes every native handle use with Close. Objects
// may keep separate state locks, but a native pointer never escapes this gate.
type nativeOperationGate struct{ mu sync.Mutex }

func (g *nativeOperationGate) enter() { g.mu.Lock() }
func (g *nativeOperationGate) leave() { g.mu.Unlock() }

// finishNativeClose preserves the close barrier without holding the native
// operation gate while already-admitted user callbacks drain.
func finishNativeClose(gate *nativeOperationGate, queue *callbackQueue) {
	gate.leave()
	queue.close()
}
