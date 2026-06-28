package main

import (
	"GoBook/internal/web"
)

func main() {
	server := web.RegisterRouters()

	server.Run(":8080")
}
