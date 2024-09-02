package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	log_v1 "github.com/dati/api/v1" // Importa los paquetes generados por Protobuf
	"github.com/dati/log"           // Importa los paquetes de log

	"github.com/gorilla/mux"
)

// Server representa el servidor HTTP que maneja las solicitudes relacionadas con los registros.
// Contiene un router para las rutas, un segmento para almacenar registros y la configuración del log.
type Server struct {
	*mux.Router              // Enrutador para manejar las rutas HTTP
	log         *log.Log     // Log general
	segment     *log.Segment // Segmento actual donde se están escribiendo/leyendo registros
	baseOffset  uint64       // Offset base del segmento actual
	nextOffset  uint64       // Siguiente offset disponible
	config      log.Config   // Configuración para el log y los segmentos
}

// NewServer crea una nueva instancia de Server y configura los segmentos de log.
// También define las rutas HTTP que serán manejadas por el servidor.
func NewServer() *Server {
	dir := "segmentos" // Define el directorio donde se almacenan los segmentos
	baseOffset := uint64(0)
	c := log.Config{
		Segment: struct {
			MaxStoreBytes uint64
			MaxIndexBytes uint64
			InitialOffset uint64
		}{
			MaxStoreBytes: 1024 * 1024, // Tamaño máximo permitido para almacenar datos
			MaxIndexBytes: 1024 * 1024, // Tamaño máximo permitido para el índice
			InitialOffset: 0,           // Offset inicial (puede ser configurado)
		},
	}

	// Inicializa el servidor con el enrutador y las configuraciones base
	s := &Server{
		Router:     mux.NewRouter(), // Configura el enrutador HTTP
		baseOffset: baseOffset,      // Establece el offset base para los segmentos
		config:     c,               // Asigna la configuración del log
	}

	// Crea un nuevo segmento para el log basado en el directorio y configuraciones especificadas
	segment, err := log.NewSegment(dir, baseOffset, c)
	if err != nil {
		fmt.Printf("Error al crear el segmento: %v\n", err)
		os.Exit(1) // Sale si no puede crear el segmento
	}
	s.segment = segment

	// Define las rutas y los manejadores para las solicitudes HTTP
	s.routes()

	return s
}

// routes define las rutas HTTP que maneja el servidor, y los métodos HTTP permitidos.
func (s *Server) routes() {
	s.HandleFunc("/record", s.appendRecord()).Methods("POST")      // Ruta para agregar un nuevo registro
	s.HandleFunc("/record/{offset}", s.getRecord()).Methods("GET") // Ruta para obtener un registro por su offset
}

// Maneja las solicitudes POST para agregar un nuevo registro al segmento.
func (s *Server) appendRecord() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rec log_v1.Record
		// Decodifica el registro desde el cuerpo de la solicitud
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			http.Error(w, "Error al decodificar la petición", http.StatusBadRequest)
			return
		}

		// Agrega el registro al segmento
		off, err := s.segment.Append(&rec)
		if err != nil {
			http.Error(w, "Error al agregar el registro", http.StatusInternalServerError)
			return
		}

		// Responde con éxito y el offset donde se agregó el registro
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "Registro creado con éxito en la posición: %d", off)
	}
}

// Para solicitudes GET para obtener un registro desde el segmento basado en su offset.
func (s *Server) getRecord() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r) // Obtiene las variables de la ruta (en este caso, el offset)
		offsetStr := vars["offset"]
		// Convierte el offset desde string a uint64
		offset, err := strconv.ParseUint(offsetStr, 10, 64)
		if err != nil {
			http.Error(w, "Offset inválido", http.StatusBadRequest)
			return
		}

		// Lee el registro desde el segmento usando el offset proporcionado
		rec, err := s.segment.Read(offset)
		if err != nil {
			http.Error(w, "Error al leer el registro", http.StatusInternalServerError)
			return
		}

		// Codifica el registro en formato JSON y lo envía como respuesta
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(rec); err != nil {
			http.Error(w, "Error al codificar la respuesta", http.StatusInternalServerError)
			return
		}
	}
}
