package log

import (
	"fmt"
	"os"
	"path"

	api "github.com/dati/api/v1"

	"google.golang.org/protobuf/proto"
)

// Segment representa un segmento del log, que contiene un store y un índice.
type Segment struct {
	store                  *Store // Almacena los registros
	index                  *Index // Índice para buscar registros en el store
	baseOffset, nextOffset uint64 // Offsets base y siguiente del segmento
	config                 Config // Configuración del segmento
}

// NewSegment crea un nuevo segmento en el directorio especificado con el offset base y configuración dados.
func NewSegment(dir string, baseOffset uint64, c Config) (*Segment, error) {
	s := &Segment{
		baseOffset: baseOffset, // Asigna el offset base
		config:     c,          // Asigna la configuración
	}
	var err error
	storeFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".store")), // Crea el archivo store
		os.O_RDWR|os.O_CREATE|os.O_APPEND,                         // Abre el archivo con permisos de lectura/escritura y creación
		0644,                                                      // Permisos del archivo
	)
	if err != nil {
		return nil, err // Retorna error si falla
	}
	if s.store, err = newStore(storeFile); err != nil {
		return nil, err // Retorna error si falla al crear el store
	}
	indexFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".index")), // Crea el archivo índice
		os.O_RDWR|os.O_CREATE,                                     // Abre el archivo con permisos de lectura/escritura y creación
		0644,                                                      // Permisos del archivo
	)
	if err != nil {
		return nil, err // Retorna error si falla
	}
	if s.index, err = newIndex(indexFile, c); err != nil {
		return nil, err // Retorna error si falla al crear el índice
	}
	if off, _, err := s.index.Read(-1); err != nil {
		s.nextOffset = baseOffset // Asigna el offset base si falla la lectura del índice
	} else {
		s.nextOffset = baseOffset + uint64(off) + 1 // Calcula el siguiente offset
	}

	return s, nil // Retorna el segmento creado
}

// Append agrega un nuevo registro al segmento.
func (s *Segment) Append(record *api.Record) (uint64, error) {
	current_offset := s.nextOffset // Asigna el offset actual
	record.Offset = current_offset // Asigna el offset al registro

	value, err := proto.Marshal(record) // Serializa el registro
	if err != nil {
		return 0, err // Retorna error si falla
	}

	_, pos, err := s.store.Append(value) // Agrega el valor serializado al store
	if err != nil {
		return 0, err // Retorna error si falla
	}
	if err = s.index.Write(
		uint32(s.nextOffset-uint64(s.baseOffset)), // Calcula el offset relativo
		pos, // Posición en el store
	); err != nil {
		return 0, err // Retorna error si falla
	}

	s.nextOffset++             // Incrementa el siguiente offset
	return current_offset, nil // Retorna el offset actual
}

// Read lee un registro del segmento basado en el offset.
func (s *Segment) Read(off uint64) (*api.Record, error) {
	_, pos, err := s.index.Read(int64(off - s.baseOffset)) // Lee la posición desde el índice
	if err != nil {
		return nil, err // Retorna error si falla
	}
	record := &api.Record{}              // Crea un nuevo registro
	record.Offset = off                  // Asigna el offset al registro
	temp_value, err := s.store.Read(pos) // Lee el valor desde el store

	if err != nil {
		return nil, err // Retorna error si falla
	}

	if err = proto.Unmarshal(temp_value, record); err != nil {
		return nil, err // Retorna error si falla la deserialización
	}

	return record, err // Retorna el registro leído
}

// IsMaxed verifica si el segmento ha alcanzado su tamaño máximo.
func (s *Segment) IsMaxed() bool {
	return s.store.size >= s.config.Segment.MaxStoreBytes || s.index.size >= s.config.Segment.MaxIndexBytes
}

// Remove elimina el segmento cerrando y eliminando sus archivos.
func (s *Segment) Remove() error {
	if err := s.Close(); err != nil {
		return err // Retorna error si falla al cerrar
	}
	if err := os.Remove(s.index.Name()); err != nil {
		return err // Retorna error si falla al eliminar el índice
	}
	if err := os.Remove(s.store.Name()); err != nil {
		return err // Retorna error si falla al eliminar el store
	}
	return nil // Retorna nil si no hay errores
}

// Close cierra el segmento cerrando el índice y el store.
func (s *Segment) Close() error {
	if err := s.index.Close(); err != nil {
		return err // Retorna error si falla al cerrar el índice
	}
	if err := s.store.Close(); err != nil {
		return err // Retorna error si falla al cerrar el store
	}
	return nil // Retorna nil si no hay errores
}

// Name devuelve el nombre del segmento basado en sus offsets.
func (s *Segment) Name() string {
	return fmt.Sprintf("%d-%d", s.baseOffset, s.nextOffset) // Formatea y retorna el nombre del segmento
}
