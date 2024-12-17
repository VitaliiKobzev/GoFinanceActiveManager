package server

type Subject interface {
	register(observer Observer)
	deregister(observer Observer)
	notifyAll()
}
