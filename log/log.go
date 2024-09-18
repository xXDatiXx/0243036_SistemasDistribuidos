package log

import (
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"

	api "github.com/dati/api/v1"
)

// Config es la estructura que contiene configuraciones específicas para el índice,
// incluyendo el tamaño máximo permitido para el store y el índice.
type Config struct {
	Segment struct {
		MaxStoreBytes uint64 // Tamaño máximo permitido para el store
		MaxIndexBytes uint64 // Tamaño máximo permitido para el índice
		InitialOffset uint64 // Offset inicial
	}
}

// Log es la estructura principal que contiene los segmentos y la configuración.
type Log struct {
	mu sync.RWMutex // Mutex para proteger el acceso concurrente

	Dir    string // Directorio donde se almacenan los segmentos
	Config Config // Configuración del log

	activeSegment *Segment   // Segmento activo actual
	segments      []*Segment // Lista de todos los segmentos
}

// NewLog crea una nueva instancia de Log.
func NewLog(dir string, c Config) (*Log, error) {
	if c.Segment.MaxStoreBytes == 0 {
		c.Segment.MaxStoreBytes = 1024 // Valor por defecto para MaxStoreBytes
	}
	if c.Segment.MaxIndexBytes == 0 {
		c.Segment.MaxIndexBytes = 1024 // Valor por defecto para MaxIndexBytes
	}
	l := &Log{
		Dir:    dir,
		Config: c,
	}

	return l, l.setup() // Configura el log y retorna la instancia
}

// setup inicializa el log configurando los segmentos existentes.
func (l *Log) setup() error {
	files, err := os.ReadDir(l.Dir) // Lee los archivos en el directorio
	if err != nil {
		return err
	}
	var baseOffsets []uint64
	for _, file := range files {
		offStr := strings.TrimSuffix(
			file.Name(),
			path.Ext(file.Name()),
		)
		off, _ := strconv.ParseUint(offStr, 10, 0) // Convierte el nombre del archivo a uint64
		baseOffsets = append(baseOffsets, off)     // Agrega el offset a la lista
	}
	sort.Slice(baseOffsets, func(i, j int) bool {
		return baseOffsets[i] < baseOffsets[j] // Ordena los offsets
	})
	for i := 0; i < len(baseOffsets); i++ {
		if err = l.NewSegment(baseOffsets[i]); err != nil {
			return err
		}
		// baseOffset contiene duplicados para índice y store, así que los saltamos
		i++
	}
	if l.segments == nil {
		if err = l.NewSegment(l.Config.Segment.InitialOffset); err != nil {
			return err
		}
	}
	return nil
}

// Append agrega un nuevo registro al segmento activo.
func (l *Log) Append(record *api.Record) (uint64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	off, err := l.activeSegment.Append(record) // Agrega el registro al segmento activo
	if err != nil {
		return 0, err
	}
	if l.activeSegment.IsMaxed() { // Verifica si el segmento ha alcanzado su tamaño máximo
		err = l.NewSegment(off + 1) // Crea un nuevo segmento
	}
	return off, err
}

// Read lee un registro del log basado en el offset.
func (l *Log) Read(off uint64) (*api.Record, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var s *Segment
	for _, segment := range l.segments {
		if segment.baseOffset <= off && off < segment.nextOffset {
			s = segment // Encuentra el segmento que contiene el offset
			break
		}
	}
	if s == nil || s.nextOffset <= off {
		return nil, fmt.Errorf("offset out of range: %d", off) // Retorna error si el offset está fuera de rango
	}
	return s.Read(off) // Lee el registro del segmento
}

// NewSegment crea un nuevo segmento y lo agrega a la lista de segmentos.
func (l *Log) NewSegment(off uint64) error {
	s, err := NewSegment(l.Dir, off, l.Config) // Crea un nuevo segmento
	if err != nil {
		return err
	}
	l.segments = append(l.segments, s) // Agrega el nuevo segmento a la lista
	l.activeSegment = s                // Establece el nuevo segmento como el activo
	return nil
}

// Close cierra todos los segmentos del log.
func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, segment := range l.segments {
		if err := segment.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Remove elimina todos los archivos del log.
func (l *Log) Remove() error {
	if err := l.Close(); err != nil {
		return err
	}
	return os.RemoveAll(l.Dir) // Elimina el directorio del log
}

// Reset reinicia el log eliminando todos los segmentos y configurándolos nuevamente.
func (l *Log) Reset() error {
	if err := l.Remove(); err != nil {
		return err
	}
	return l.setup() // Configura nuevamente el log
}

// LowestOffset retorna el offset más bajo en el log.
func (l *Log) LowestOffset() (uint64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.segments[0].baseOffset, nil // Retorna el offset base del primer segmento
}

// HighestOffset retorna el offset más alto en el log.
func (l *Log) HighestOffset() (uint64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	off := l.segments[len(l.segments)-1].nextOffset // Obtiene el siguiente offset del último segmento
	if off == 0 {
		return 0, nil
	}
	return off - 1, nil // Retorna el offset más alto
}

// Truncate elimina los segmentos cuyo offset es menor al especificado.
func (l *Log) Truncate(lowest uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	var segments []*Segment
	for _, s := range l.segments {
		if s.nextOffset <= lowest+1 {
			if err := s.Remove(); err != nil {
				return err
			}
			continue
		}
		segments = append(segments, s) // Mantiene los segmentos que no se eliminan
	}
	l.segments = segments
	return nil
}

// Reader retorna un lector que permite leer todos los registros en el log.
func (l *Log) Reader() io.Reader {
	l.mu.RLock()
	defer l.mu.RUnlock()
	readers := make([]io.Reader, len(l.segments))
	for i, segment := range l.segments {
		readers[i] = &originReader{segment.store, 0} // Crea un lector para cada segmento
	}
	return io.MultiReader(readers...) // Combina todos los lectores en uno solo
}

// originReader es un lector que lee desde el inicio del store.
type originReader struct {
	*Store
	off int64 // Offset actual del lector
}

// Read lee datos desde el store en el offset actual.
func (o *originReader) Read(p []byte) (int, error) {
	n, err := o.ReadAt(p, o.off) // Lee datos desde el offset actual
	o.off += int64(n)            // Actualiza el offset
	return n, err
}
