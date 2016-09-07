package container

import "sync"

// Constructor is a function that is invoked to create the value for a requested key.
type Constructor func(*Definer) (interface{}, error)

// Definer is used to execute a Definer.
type Definer struct {
	Container     *Container
	key           string
	value         interface{}
	constructor   Constructor
	isConstructed bool
	deferredCalls []func()
	mutex         sync.RWMutex
}

// Get a value from the container.
func (def *Definer) Get(key string) interface{} {
	return def.Container.Get(key)
}

// TryGet a value from the container.
func (def *Definer) TryGet(key string) (interface{}, error) {
	return def.Container.TryGet(key)
}

// Defer a function to be invoked when the container is closed.
func (def *Definer) Defer(fn func()) {
	def.deferredCalls = append(def.deferredCalls, fn)
}

func (def *Definer) close() {
	def.mutex.Lock()
	defer def.mutex.Unlock()

	for _, fn := range def.deferredCalls {
		defer fn()
	}
}

func (def *Definer) get() (interface{}, error) {
	if def.isConstructed {
		return def.value, nil
	}

	def.mutex.Lock()
	defer def.mutex.Unlock()

	if !def.isConstructed {
		value, err := def.constructor(def)
		if err != nil {
			def.deferredCalls = nil
			return nil, err
		}

		def.value = value
		def.isConstructed = true
	}

	return def.value, nil
}
