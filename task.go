package gpool

type RunTaskFunc func()
type CallTaskFunc func() interface{}

type item interface {
	run()
	discard()
	abortWithErr(err error)
}

type Result interface {
	Error() error
	Done() <-chan interface{}
}

type task interface {
	item
	Result
}

type RunTask struct {
	fn   func()
	done chan interface{}
	err  error
}

func (t *RunTask) run() {
	t.fn()
	close(t.done)
}

func (t *RunTask) discard() {
	t.err = ErrTaskDiscard
	close(t.done)
}

func (t *RunTask) abortWithErr(err error) {
	t.err = err
	close(t.done)
}

func (t *RunTask) Error() error {
	return t.err
}

func (t *RunTask) Done() <-chan interface{} {
	return t.done
}

type CallTask struct {
	fn   func() interface{}
	done chan interface{}
	err  error
}

func (t *CallTask) run() {
	t.done <- t.fn()
	close(t.done)
}

func (t *CallTask) discard() {
	t.err = ErrTaskDiscard
	close(t.done)
}

func (t *CallTask) abortWithErr(err error) {
	t.err = err
	close(t.done)
}

func (t *CallTask) Done() <-chan interface{} {
	return t.done
}

func (t *CallTask) Error() error {
	return t.err
}
