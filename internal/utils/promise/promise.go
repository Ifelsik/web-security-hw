package promise

type Result[E any] struct {
	Value E
	Err   error
}

func Promise[E any](task func() (E, error)) chan Result[E] {
	resultCh := make(chan Result[E], 1)

	go func() {
		result, err := task()
		resultCh <- Result[E]{result, err}
		close(resultCh)
	}()

	return resultCh
}
