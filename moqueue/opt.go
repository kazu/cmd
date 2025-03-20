package moqueue

type OptFunc[T any] func(*T) OptFunc[T]
