package api // Nombre del paquete

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/mux" // Importar el paquete mux, nos ayuda a manejar rutas
)

// Definición del Record, que representa cada registro en el log
type Record struct {
	Value  []byte `json:"value"`  // Valor del registro en bytes
	Offset uint64 `json:"offset"` // Offset del registro
}

// Definición del Log, que representa el commit log
type Log struct {
	mu      sync.Mutex // Mutex para asegurar acceso concurrente seguro
	records []Record   // Slice que almacena los registros
}

// Definición de un servidor que manejará los registros
type Server struct {
	*mux.Router // Agregar un router
	log         *Log
}

func NewServer() *Server { // Crea y retorna un nuevo servidor
	s := &Server{
		Router: mux.NewRouter(),
		log:    &Log{records: []Record{}}, // Inicializa el log
	}

	s.routes() // Configura las rutas del servidor
	return s
}

func (s *Server) routes() { // Define las rutas que el servidor manejará
	s.HandleFunc("/record", s.appendRecord()).Methods(http.MethodPost)      // Endpoint para agregar un nuevo record
	s.HandleFunc("/record/{offset}", s.getRecord()).Methods(http.MethodGet) // Endpoint para obtener un record por su offset
}

func (s *Server) appendRecord() http.HandlerFunc { // Método para manejar la creación de un nuevo record
	return func(w http.ResponseWriter, r *http.Request) {
		var rec Record

		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil { // Decodifica el JSON a una estructura Record
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		s.log.mu.Lock() // Asegura acceso exclusivo al log
		defer s.log.mu.Unlock()

		rec.Offset = uint64(len(s.log.records))    // Calcula el offset basado en la longitud actual de los registros
		s.log.records = append(s.log.records, rec) // Añade el nuevo record al log

		if err := json.NewEncoder(w).Encode(rec); err != nil { // Codifica y envía el record como respuesta
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s *Server) getRecord() http.HandlerFunc { // Método para manejar la obtención de un record por su offset
	return func(w http.ResponseWriter, r *http.Request) {
		offsetStr := mux.Vars(r)["offset"] // Obtiene el offset de los parámetros de la URL

		var offset uint64
		if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil {
			http.Error(w, "Invalid offset", http.StatusBadRequest)
			return
		}

		s.log.mu.Lock() // Asegura acceso exclusivo al log
		defer s.log.mu.Unlock()

		if offset >= uint64(len(s.log.records)) {
			http.Error(w, "Record not found", http.StatusNotFound)
			return
		}

		record := s.log.records[offset] // Obtiene el record basado en el offset

		if err := json.NewEncoder(w).Encode(record); err != nil { // Codifica y envía el record como respuesta
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
