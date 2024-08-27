package api // Nombre del paquete

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"os"

	"github.com/gorilla/mux" // Importar el paquete mux, nos ayuda a manejar rutas
	"github.com/dati/store" // Importar el paquete store.go
	"github.com/dati/indice" // Importar el paquete index.go

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
    log *store.Store // Usa estructura Store
	idx *index.Index // Agregar un índice
}

func NewServer() *Server {
    // Inicializa el archivo de registros
    logFile, err := os.OpenFile("store.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
    if err != nil {
        panic(err)
    }

    st, err := store.NewStore(logFile)
    if err != nil {
        panic(err)
    }

    // Inicializa el archivo de índice
    indexFile, err := os.OpenFile("store.index", os.O_CREATE|os.O_RDWR, 0666)
    if err != nil {
        panic(err)
    }

    idx, err := index.NewIndex(indexFile)
    if err != nil {
        panic(err)
    }

    s := &Server{
        Router: mux.NewRouter(),
        log:    st,
        idx:    idx,
    }

    s.routes()
    return s
}

func (s *Server) appendRecord() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var rec store.Record

        if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }

        // Escribe en el store
        n, pos, err := s.log.Append(rec.Value)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        // Actualiza el índice con el nuevo registro
        if err := s.idx.Write(uint32(pos), n); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        rec.Offset = pos

        if err := json.NewEncoder(w).Encode(rec); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    }
}

func (s *Server) getRecord() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        offsetStr := mux.Vars(r)["offset"]

        var offset uint64
        if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil {
            http.Error(w, "Invalid offset", http.StatusBadRequest)
            return
        }

        // Lee la posición desde el índice
        _, pos, err := s.idx.Read(int64(offset))
        if err != nil {
            http.Error(w, "Record not found", http.StatusNotFound)
            return
        }

        // Usa la posición para leer el registro desde el store
        rec, err := s.log.Read(pos)
        if err != nil {
            http.Error(w, "Record not found", http.StatusNotFound)
            return
        }

        if _, err := w.Write(rec); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    }
}

