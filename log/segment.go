package log

import (
	"fmt"
	"io"
	"os"
	"path"

	log_v1 "github.com/dati/api/v1" // Importa el paquete generado automáticamente por Protobuf

	"google.golang.org/protobuf/proto"
)

// Representa un segmento dentro del log.
// Cada segmento contiene un store para almacenar registros y un index para
// mapear offsets a posiciones dentro del store.
type Segment struct {
	store                  *Store // Almacena los datos en formato binario
	index                  *Index // Índice que mapea offsets a posiciones dentro del store
	baseOffset, nextOffset uint64 // baseOffset es el offset inicial del segmento, nextOffset es el siguiente offset disponible
	config                 Config // Configuración del segmento (tamaños máximos de store e index)
}

// Crea un nuevo segmento con el baseOffset dado y lo inicializa.
// Los archivos .store y .index se crean y se inicializan en el directorio segments, especificado
// en server.go.
func NewSegment(dir string, baseOffset uint64, c Config) (*Segment, error) {
	s := &Segment{
		baseOffset: baseOffset,
		config:     c,
	}

	var err error

	// Abre o crea el archivo store correspondiente a este segmento.
	storeFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".store")),
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, err
	}
	if s.store, err = NewStore(storeFile); err != nil {
		return nil, err
	}

	// Abre o crea el archivo index correspondiente a este segmento.
	indexFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".index")),
		os.O_RDWR|os.O_CREATE,
		0644,
	)
	if err != nil {
		return nil, err
	}
	if s.index, err = NewIndex(indexFile, c); err != nil {
		return nil, err
	}

	// Si el índice tiene registros, ajusta nextOffset para continuar desde el último offset.
	if off, _, err := s.index.Read(-1); err != nil {
		s.nextOffset = baseOffset
	} else {
		s.nextOffset = baseOffset + uint64(off) + 1
	}

	return s, nil
}

// Append agrega un nuevo record al segmento.
// Serializa el record usando Protobuf, lo almacena en el store,
// y luego actualiza el índice con la posición del record en el store.
func (s *Segment) Append(record *log_v1.Record) (uint64, error) {
	// Verifica si el segmento ha alcanzado su tamaño máximo.
	if s.IsMaxed() {
		return 0, io.EOF
	}

	// Asigna el próximo offset disponible al record.
	offset := s.nextOffset
	record.Offset = offset

	// Serializa el record a formato binario.
	p, err := proto.Marshal(record)
	if err != nil {
		return 0, err
	}

	// Almacena el record en el store y obtiene su posición.
	pos, _, err := s.store.Append(p)
	if err != nil {
		return 0, err
	}

	// Actualiza el índice con el offset relativo y la posición en el store.
	if err = s.index.Write(
		uint32(s.nextOffset-uint64(s.baseOffset)),
		pos,
	); err != nil {
		return 0, err
	}

	// Incrementa el nextOffset para el siguiente record.
	s.nextOffset++
	return offset, nil
}

// Read lee un record desde el segmento basado en el offset dado.
// Usa el índice para encontrar la posición correcta en el store y luego
// deserializa el record desde esa posición.
func (s *Segment) Read(off uint64) (*log_v1.Record, error) {
	// Encuentra la posición del record en el store usando el índice.
	_, pos, err := s.index.Read(int64(off - s.baseOffset))
	if err != nil {
		return nil, err
	}

	// Lee el record desde la posición en el store.
	p, err := s.store.Read(pos)
	if err != nil {
		return nil, err
	}

	// Deserializa el record desde el formato binario.
	record := &log_v1.Record{}
	err = proto.Unmarshal(p, record)
	return record, err
}

// IsMaxed verifica si el segmento ha alcanzado su tamaño máximo.
// Revisa tanto el tamaño del store como el del índice para determinar si el
// segmento está "lleno" y ya no puede aceptar más registros.
func (s *Segment) IsMaxed() bool {
	// Verifica si el tamaño del store ha alcanzado su límite máximo
	if s.store.size >= s.config.Segment.MaxStoreBytes {
		return true
	}

	// Verifica si el tamaño del índice ha alcanzado su límite máximo
	if s.index.size >= s.config.Segment.MaxIndexBytes {
		return true
	}

	// Si ninguno de los dos ha alcanzado su límite, devuelve false
	return false
}

// Remove elimina los archivos del segmento del sistema de archivos.
// Primero cierra los archivos, luego los elimina.
func (s *Segment) Remove() error {
	// Cierra los archivos del segmento.
	if err := s.Close(); err != nil {
		return err
	}

	// Elimina los archivos del índice y el store.
	if err := os.Remove(s.index.Name()); err != nil {
		return err
	}
	return os.Remove(s.store.Name())
}

// Close cierra los archivos del segmento, asegurando que todos los cambios
// se escriban en disco antes de cerrar.
func (s *Segment) Close() error {
	if err := s.store.Close(); err != nil {
		return err
	}
	return s.index.Close()
}
