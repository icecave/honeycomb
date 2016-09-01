package di

// Container stores wires together the root-level application dependencies.
type Container struct {
	values  map[string]interface{}
	closers []func() error
}

// Close cleans up any resources used by dependencies.
func (con *Container) Close() {
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
	close func(interface{}) error,
) interface{} {
	value, ok := con.values[name]

	if !ok {
		if con.values == nil {
			con.values = map[string]interface{}{}
		} else {
			value, ok = con.values[name]
			if ok {
				return value
			}
		}

		var err error
		value, err = initialize()
		if err != nil {
			panic(err)
		}

		con.values[name] = value

		if close != nil {
			con.closers = append(
				con.closers,
				func() error {
					return close(value)
				},
			)
		}
	}

	return value
}
