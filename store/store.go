package store

import (
    "bufio"
    "encoding/binary"
    "os"
    "sync"
)

// Definición del Record, que representa cada registro en el log
type Record struct {
	Value  []byte `json:"value"`  // Valor del registro en bytes
	Offset uint64 `json:"offset"` // Offset del registro
}

// Definir el encoder que usaremos para el archivo y el ancho de cada registro
var (
    enc      = binary.BigEndian
    lenWidth = 8 // Ancho en bytes para espaciar registros
)

// Estructura que representa el almacenamiento (Store)
type Store struct {
    *os.File         // El archivo donde guardamos los registros
    mu      sync.Mutex // Mutex para acceso concurrente seguro
    buf     *bufio.Writer // Buffer para escribir datos antes de enviarlos al archivo
    size    uint64       // Tamaño del archivo en bytes
}

// Constructor para la estructura `store`
func NewStore(f *os.File) (*Store, error) {
    size, err := f.Seek(0, os.SEEK_END)
    if err != nil {
        return nil, err
    }
    return &Store{
        File: f,
        size: uint64(size),
        buf:  bufio.NewWriter(f),
    }, nil
}


// Función para agregar datos (append) al store
func (s *Store) Append(p []byte) (n uint64, pos uint64, err error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    pos = s.size
    if err := binary.Write(s.buf, enc, uint64(len(p))); err != nil { // Escribir el tamaño de los datos
        return 0, 0, err
    }
    n1, err := s.buf.Write(p) // Escribir los datos en sí
    if err != nil {
        return 0, 0, err
    }

    n = uint64(n1) + uint64(lenWidth) // Convertir lenWidth a uint64
    s.size += n

    return n, pos, nil
}

// Función para leer datos desde el store
func (s *Store) Read(pos uint64) ([]byte, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if err := s.buf.Flush(); err != nil { // Asegurarse de que los datos en el buffer se escriben en el archivo
        return nil, err
    }

    size := make([]byte, lenWidth)
    if _, err := s.File.ReadAt(size, int64(pos)); err != nil {
        return nil, err
    }
    p := make([]byte, enc.Uint64(size))
    if _, err := s.File.ReadAt(p, int64(pos+uint64(lenWidth))); err != nil { // Convertir lenWidth a uint64
        return nil, err
    }

    return p, nil
}

// Función helper para leer datos en una posición específica
func (s *Store) ReadAt(p []byte, off int64) (int, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if err := s.buf.Flush(); err != nil {
        return 0, err
    }
    return s.File.ReadAt(p, off)
}

// Función para cerrar el archivo y persistir datos
func (s *Store) Close() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    err := s.buf.Flush() // Escribir cualquier dato pendiente en el buffer
    if err != nil {
        return err
    }
    return s.File.Close() // Cerrar el archivo
}
