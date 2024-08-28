package api

import (
	"log"
	"net/http"
	"os"

	index "github.com/dati/indice"
	"github.com/dati/store"

	"github.com/gorilla/mux"
)

// Definición de un servidor que manejará los registros
type Server struct {
	*mux.Router              // Agregar un router
	log         *store.Store // Usa estructura Store
	idx         *index.Index // Agregar un índice
}

func NewServer() *Server {
	// Inicializa el archivo de registros
	logFile, err := os.OpenFile("store.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("Error al abrir el archivo de registros: %v", err)
	}

	st, err := store.NewStore(logFile)
	if err != nil {
		log.Fatalf("Error al inicializar el store: %v", err)
	}

	// Inicializa el archivo de índice
	indexFile, err := os.OpenFile("store.index", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("Error al abrir el archivo de índice: %v", err)
	}

	idx, err := index.NewIndex(indexFile)
	if err != nil {
		log.Fatalf("Error al inicializar el índice: %v", err)
	}

	s := &Server{
		Router: mux.NewRouter(),
		log:    st,
		idx:    idx,
	}

	s.routes()
	return s
}

func (s *Server) routes() {
	s.HandleFunc("/your-endpoint", s.handleYourEndpoint()).Methods("GET")
}

func (s *Server) handleYourEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Your handler logic here
		w.Write([]byte("Hello, world!"))
	}
}

func main() {
	s := NewServer()
	log.Fatal(http.ListenAndServe(":8080", s))
}
