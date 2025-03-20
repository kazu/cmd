package moqueue

import "github.com/kazu/cmd/ext"

type MQueue[T any] struct {
	buf []T
	MQueueCnf
}

type MQueueCnf struct {
	reader int
	writer int
}

func Reader(rcnt int) ext.OptFunc[MQueueCnf] {

	return func(arg *MQueueCnf) ext.OptFunc[MQueueCnf] {
		old := arg.reader
		arg.reader = rcnt
		return Reader(old)
	}
}

func Writer(rcnt int) ext.OptFunc[MQueueCnf] {

	return func(arg *MQueueCnf) ext.OptFunc[MQueueCnf] {
		old := arg.writer
		arg.writer = rcnt
		return Writer(old)
	}
}

func New[T any](opts ...ext.OptFunc[MQueueCnf]) *MQueue[T] {

	mq := ext.NewWithDefault[MQueue[T]]()

	mq.reader = 1
	mq.reader = 2

	prev := ext.HandleOpt(&mq.MQueueCnf, opts...)
	if mq.writer > mq.reader {
		goto ERROR
	}
	if mq.reader%2 != 0 {
		goto ERROR
	}
	if mq.reader%mq.writer != 0 {
		goto ERROR
	}

	return mq

ERROR:

	ext.HandleOpt(&mq.MQueueCnf, prev)
	return mq

}
