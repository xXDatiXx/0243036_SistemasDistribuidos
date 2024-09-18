package log

import (
	"bufio"
	"encoding/binary"
	"os"
	"sync"
)

var (
	enc = binary.BigEndian // Define el orden de bytes como BigEndian
)

const (
	lenWidth = 8 // Define el ancho del campo de longitud en bytes
)

// Store representa el almacenamiento de registros en un archivo.
type Store struct {
	*os.File               // Archivo donde se almacenan los registros
	mu       sync.Mutex    // Mutex para proteger el acceso concurrente
	buf      *bufio.Writer // Buffer para escritura eficiente
	size     uint64        // Tamaño actual del archivo en bytes
}

// newStore crea una nueva instancia de Store a partir de un archivo dado.
func newStore(f *os.File) (*Store, error) {
	file_info, err := f.Stat() // Obtiene información del archivo
	if err != nil {
		return nil, err // Retorna error si falla
	}
	return &Store{
		File: f,                        // Asigna el archivo al Store
		buf:  bufio.NewWriter(f),       // Crea un nuevo buffer para el archivo
		size: uint64(file_info.Size()), // Asigna el tamaño del archivo al Store
	}, nil // Retorna la instancia de Store
}

// Read lee un registro desde el Store basado en el offset dado.
func (s *Store) Read(in uint64) (out []byte, err error) {
	if err := s.buf.Flush(); err != nil { // Vacía el buffer al archivo
		return nil, err // Retorna error si falla
	}

	value_size_bytes := make([]byte, lenWidth) // Crea un buffer para el tamaño del valor

	if _, err := s.File.ReadAt(value_size_bytes, int64(in)); err != nil { // Lee el tamaño del valor desde el archivo
		return nil, err // Retorna error si falla
	}

	value_size := enc.Uint64(value_size_bytes) // Decodifica el tamaño del valor

	value := make([]byte, value_size) // Crea un buffer para el valor

	if _, err := s.File.ReadAt(value, int64(in+lenWidth)); err != nil { // Lee el valor desde el archivo
		return nil, err // Retorna error si falla
	}

	return value, nil // Retorna el valor leído
}

// ReadAt lee datos desde el Store en una posición específica.
func (s *Store) ReadAt(p []byte, off int64) (int, error) {
	s.mu.Lock()                           // Bloquea el mutex para acceso exclusivo
	defer s.mu.Unlock()                   // Desbloquea el mutex al salir de la función
	if err := s.buf.Flush(); err != nil { // Vacía el buffer al archivo
		return 0, err // Retorna error si falla
	}
	return s.File.ReadAt(p, int64(off)) // Lee datos desde el archivo en la posición especificada
}

// Append agrega un nuevo registro al Store.
func (s *Store) Append(value []byte) (bytes uint64, off uint64, err error) {
	s.mu.Lock()         // Bloquea el mutex para acceso exclusivo
	defer s.mu.Unlock() // Desbloquea el mutex al salir de la función

	if err := s.buf.Flush(); err != nil { // Vacía el buffer al archivo
		return 0, 0, err // Retorna error si falla
	}

	off = s.size                                                         // Asigna el offset actual
	if err := binary.Write(s.buf, enc, uint64(len(value))); err != nil { // Escribe el tamaño del valor en el buffer
		return 0, 0, err // Retorna error si falla
	}
	if err := binary.Write(s.buf, enc, value); err != nil { // Escribe el valor en el buffer
		return 0, 0, err // Retorna error si falla
	}

	s.size += lenWidth + uint64(len(value)) // Incrementa el tamaño del Store

	return uint64(lenWidth) + uint64(len(value)), off, nil // Retorna el número de bytes escritos y el offset
}

// Remove elimina el archivo del Store.
func (s *Store) Remove() error {
	if err := s.Close(); err != nil { // Cierra el Store
		return err // Retorna error si falla
	}
	return os.Remove(s.Name()) // Elimina el archivo y retorna error si falla
}

// Close cierra el Store vaciando el buffer y cerrando el archivo.
func (s *Store) Close() error {
	if err := s.buf.Flush(); err != nil { // Vacía el buffer al archivo
		return err // Retorna error si falla
	}
	return s.File.Close() // Cierra el archivo y retorna error si falla
}
