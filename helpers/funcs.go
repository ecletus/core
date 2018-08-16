package helpers

func CheckReturnError(funcs ...func()(key string, err error)) (name string, err error) {
	for _, f := range funcs {
		name, err = f()
		if err != nil {
			return name, err
		}
	}
	return "", nil
}
