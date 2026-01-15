package routine

func Recover(cleanups ...func(r interface{})) {
	if r := recover(); r != nil {
		for _, cleanup := range cleanups {
			cleanup(r)
		}
	}
}
