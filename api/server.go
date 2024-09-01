package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	index "github.com/dati/log" // Import the package that contains the index type
	store "github.com/dati/log"

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
		fmt.Printf("Error al abrir el archivo de registros: %v\n", err)
		os.Exit(1)
	}

	st, err := store.NewStore(logFile)
	if err != nil {
		fmt.Printf("Error al inicializar el store: %v\n", err)
		os.Exit(1)
	}

	// Inicializa el archivo de índice
	indexFile, err := os.OpenFile("store.index", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Printf("Error al abrir el archivo de índice: %v\n", err)
		os.Exit(1)
	}

	// Crear la configuración para el índice con un tamaño mayor
	config := index.Config{}
	config.Segment.MaxIndexBytes = 10 * 1024 * 1024 // 10 MB de espacio para el índice

	// Pasar el archivo de índice y la configuración a NewIndex
	idx, err := index.NewIndex(indexFile, config)
	if err != nil {
		fmt.Printf("Error al inicializar el índice: %v\n", err)
		os.Exit(1)
	}

	s := &Server{
		Router: mux.NewRouter(),
		log:    st,
		idx:    idx,
	}

	s.routes() // Configura las rutas
	return s
}

// Define las rutas que el servidor manejará
func (s *Server) routes() {
	s.HandleFunc("/record", s.appendRecord()).Methods(http.MethodPost)      // Endpoint para agregar un nuevo record
	s.HandleFunc("/record/{offset}", s.getRecord()).Methods(http.MethodGet) // Endpoint para obtener un record por su offset
}

// Método para manejar la creación de un nuevo registro
func (s *Server) appendRecord() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rec store.Record
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			http.Error(w, "Error al decodificar la petición", http.StatusBadRequest)
			return
		}

		// Guardar el registro en el store y actualizar el índice
		_, pos, err := s.log.Append(rec.Value)
		if err != nil {
			http.Error(w, "Error al guardar el registro", http.StatusInternalServerError)
			return
		}

		if err := s.idx.Write(uint32(rec.Offset), pos); err != nil {
			http.Error(w, "Error al actualizar el índice", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "Registro creado con éxito en la posición: %d", pos)
	}
}

// Método para manejar la obtención de un registro por su offset
func (s *Server) getRecord() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		offsetStr := vars["offset"]
		offset, err := strconv.ParseInt(offsetStr, 10, 64)
		if err != nil {
			http.Error(w, "Offset inválido", http.StatusBadRequest)
			return
		}

		// Leer desde el índice
		off, pos, err := s.idx.Read(offset)
		if err != nil {
			http.Error(w, "Error al leer del índice", http.StatusInternalServerError)
			return
		}

		// Leer el registro desde el store usando la posición obtenida del índice
		data, err := s.log.Read(pos)
		if err != nil {
			http.Error(w, "Error al leer el registro", http.StatusInternalServerError)
			return
		}

		// Devolver el registro como JSON
		rec := store.Record{Value: data, Offset: uint64(off)}
		if err := json.NewEncoder(w).Encode(rec); err != nil {
			http.Error(w, "Error al codificar la respuesta", http.StatusInternalServerError)
			return
		}
	}
}
