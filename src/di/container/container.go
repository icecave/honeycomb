package container

import (
	"fmt"
	"os"
	"sync"
)

// Container is a lazy loading dependency injection container.
type Container struct {
	defs  map[string]*Definer
	mutex sync.RWMutex
}

// Define adds a new entry to the container, which is lazily set the first time
// it is requested.
func (con *Container) Define(key string, constructor Constructor) {
	con.mutex.Lock()
	defer con.mutex.Unlock()

	if con.defs == nil {
		con.defs = map[string]*Definer{}
	}

	con.defs[key] = &Definer{
		Container:   con,
		key:         key,
		constructor: constructor,
	}
}

// DefineEnv adds a new environment variable to the container.
func (con *Container) DefineEnv(key string, defaultValue string) {
	con.Define(key, func(d *Definer) (interface{}, error) {
		if value := os.Getenv(key); value != "" {
			return value, nil
		}

		return defaultValue, nil
	})
}

// Get retreives an entry from the container.
func (con *Container) Get(key string) interface{} {
	entry, err := con.TryGet(key)

	if err != nil {
		panic(err)
	}

	return entry
}

// TryGet attempst to retreive an entry from the container.
func (con *Container) TryGet(key string) (interface{}, error) {
	con.mutex.RLock()
	def, ok := con.defs[key]
	con.mutex.RUnlock()

	if ok {
		return def.get()
	}

	return nil, fmt.Errorf("no definition for '%s' in container", key)
}

// Close the container, invoking any deferred functions configured during the
// definition phase.
func (con *Container) Close() {
	con.mutex.Lock()
	defer con.mutex.Unlock()

	for _, def := range con.defs {
		def.close()
	}

	con.defs = nil
}
