package log

import (
	"bufio"
	"encoding/binary"
	"os"
	"sync"
)

// Record representa cada registro almacenado en el log.
// Contiene un valor en bytes y un offset que indica la posición del registro en el log.
type Record struct {
	Value  []byte `json:"value"`  // Valor del registro en bytes
	Offset uint64 `json:"offset"` // Offset del registro dentro del log
}

// Variable global para la codificación y el tamaño de cada registro en bytes.
var (
	enc      = binary.BigEndian // Usamos codificación BigEndian para escribir datos binarios en el archivo.
	lenWidth = 8                // Ancho en bytes utilizado para almacenar el tamaño de cada registro.
)

// Store representa el almacenamiento persistente donde se guardan los registros.
// Está asociado con un archivo en el sistema de archivos.
type Store struct {
	*os.File               // Archivo donde se almacenan los registros
	mu       sync.Mutex    // Mutex para asegurar acceso concurrente seguro
	buf      *bufio.Writer // Buffer de escritura para mejorar el rendimiento al escribir en el archivo
	size     uint64        // Tamaño actual del archivo en bytes
}

// NewStore es un constructor que inicializa una nueva instancia de Store.
// Abre el archivo asociado y determina su tamaño actual.
func NewStore(f *os.File) (*Store, error) {
	// Seek mueve el puntero del archivo al final para determinar su tamaño.
	size, err := f.Seek(0, os.SEEK_END)
	if err != nil {
		return nil, err
	}
	return &Store{
		File: f,
		size: uint64(size),       // Establece el tamaño del archivo
		buf:  bufio.NewWriter(f), // Inicializa el buffer de escritura
	}, nil
}

// Append agrega un nuevo registro al store.
// Escribe el tamaño del registro seguido por el registro.
// Devuelve el número de bytes escritos y la posición donde se almacenó el registro.
func (s *Store) Append(p []byte) (n uint64, pos uint64, err error) {
	s.mu.Lock() // Asegura que solo una goroutine pueda escribir en el store a la vez.
	defer s.mu.Unlock()

	pos = s.size // Establece la posición actual como el punto donde se escribirá el nuevo registro.
	// Escribe el tamaño del registro en el buffer.
	if err := binary.Write(s.buf, enc, uint64(len(p))); err != nil {
		return 0, 0, err
	}
	// Escribe el registro en sí en el buffer.
	n1, err := s.buf.Write(p)
	if err != nil {
		return 0, 0, err
	}

	// El tamaño total incluye tanto el tamaño del registro como los datos.
	n = uint64(n1) + uint64(lenWidth)
	s.size += n // Actualiza el tamaño del store

	return n, pos, nil
}

// Read lee un registro del store desde una posición dada.
// Devuelve los datos leídos como un slice de bytes.
func (s *Store) Read(pos uint64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Asegura que cualquier dato pendiente en el buffer se escriba en el archivo.

	size := make([]byte, lenWidth)
	// Lee el tamaño del registro desde la posición especificada en el store.
	if _, err := s.File.ReadAt(size, int64(pos)); err != nil {
		return nil, err
	}
	// Lee el registro completo basado en el tamaño obtenido.
	p := make([]byte, enc.Uint64(size))
	if _, err := s.File.ReadAt(p, int64(pos+uint64(lenWidth))); err != nil {
		return nil, err
	}

	return p, nil
}

// ReadAt es una función que permite leer directamente del store desde una posición específica.
// Es útil cuando se necesita leer datos en ubicaciones específicas sin procesar todo el registro.
func (s *Store) ReadAt(p []byte, off int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Asegura que cualquier dato pendiente en el buffer se escriba en el archivo antes de leer.
	if err := s.buf.Flush(); err != nil {
		return 0, err
	}
	// Lee los datos desde la posición especificada en el archivo.
	return s.File.ReadAt(p, off)
}

// Close cierra el store, asegurando que todos los datos se escriban en el archivo antes de cerrarlo.
// También libera cualquier recurso asociado con el archivo.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Escribe cualquier dato pendiente en el buffer al archivo.
	if err := s.buf.Flush(); err != nil {
		return err
	}
	// Cierra el archivo para liberar recursos.
	return s.File.Close()
}
