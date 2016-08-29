package di

import "sync"

// Container stores wires together the root-level application dependencies.
type Container struct {
	values  map[string]interface{}
	closers []func() error
	mutex   sync.RWMutex
}

// Close cleans up any resources used by dependencies.
func (con *Container) Close() {
	con.mutex.Lock()
	defer con.mutex.Unlock()

	closers := con.closers
	con.values = nil
	con.closers = nil

	for _, fn := range closers {
		err := fn()
		if err != nil {
			panic(err)
		}
	}
}

func (con *Container) get(
	name string,
	initialize func() (interface{}, error),
	close func() error,
) interface{} {
	con.mutex.RLock()
	value, ok := con.values[name]
	con.mutex.RUnlock()

	if !ok {
		con.mutex.Lock()
		defer con.mutex.Unlock()

		if con.values == nil {
			con.values = map[string]interface{}{}
		}

		var err error
		value, err = initialize()
		if err != nil {
			panic(err)
		}

		con.values[name] = value

		if close != nil {
			con.closers = append(con.closers, close)
		}
	}

	return value
}
