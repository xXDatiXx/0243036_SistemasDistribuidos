// localhost:8080

package main

import (
	"fmt"
	"net"

	api "github.com/dati/api/v1"
	log "github.com/dati/log"
	"github.com/dati/server"
	"google.golang.org/grpc"
)

func main() {
	// Escuchar en el puerto 8080
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("Error al iniciar el listener: %v", err)
	}

	// Crear un nuevo CommitLog
	commitLog, err := log.NewLog("/tmp/commitlog", log.Config{})
	if err != nil {
		fmt.Printf("Error al inicializar el commit log: %v", err)
	}

	// Inicializar el servidor gRPC
	grpcServer, err := server.NewGRPCServer(commitLog)
	if err != nil {
		fmt.Printf("Error al inicializar el servidor gRPC: %v", err)
	}

	// Crear una nueva instancia del servidor gRPC
	s := grpc.NewServer()
	api.RegisterLogServer(s, grpcServer)

	fmt.Println("Servidor gRPC escuchando en el puerto 8080...")

	// Iniciar el servidor gRPC
	if err := s.Serve(listener); err != nil {
		fmt.Printf("Error al iniciar el servidor gRPC: %v", err)
	}
}
