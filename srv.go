package http

import (
	"github.com/Nixson/http/server"
)

func InitSever() {
	server.InitServer()
}
func RunServer() {
	server.Run()
}

func InitController(name string, controller *server.ContextInterface) {
	server.InitController(name, controller)
}
