package server

type Observer interface {
	Update(string)
	GetID() string
}
