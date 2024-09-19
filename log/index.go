package log

//que permite mapear las posiciones de los registros a las posiciones en el archivo físico. 
// El índice facilita la búsqueda rápida de registros en el almacenamiento.

import (
	"io"
	"os"

	"github.com/tysonmote/gommap"
)

// Variables que definen el ancho de los campos en el índice.
// offWidth es el tamaño del offset (4 bytes), posWidth es el tamaño de la posición (8 bytes),
// y entWidth es el tamaño total de una entrada en el índice.
var (
	offWidth uint64 = 4                   // Tamaño del offset en bytes
	posWidth uint64 = 8                   // Tamaño de la posición en bytes
	entWidth        = offWidth + posWidth // Tamaño total de una entrada en el índice
)

// Index representa el índice de un segmento, que mapea offsets a posiciones en el store.
type Index struct {
	file *os.File    // Archivo en el cual se almacena el índice
	mmap gommap.MMap // Mapeo de memoria para acceder al archivo del índice
	size uint64      // Tamaño actual del índice en bytes
}

// NewIndex crea un nuevo índice a partir de un archivo dado y configura el mapeo a memoria.
// Devuelve una instancia de Index o un error si falla.
func newIndex(f *os.File, c Config) (*Index, error) {
	idx := &Index{
		file: f, // Asigna el archivo al índice
	}
	fi, err := os.Stat(f.Name()) // Obtiene información del archivo
	if err != nil {
		return nil, err // Retorna error si falla
	}
	idx.size = uint64(fi.Size()) // Asigna el tamaño del archivo al índice
	if err = os.Truncate(
		f.Name(), int64(c.Segment.MaxIndexBytes), // Trunca el archivo al tamaño máximo permitido
	); err != nil {
		return nil, err // Retorna error si falla
	}
	if idx.mmap, err = gommap.Map(
		idx.file.Fd(),                      // Mapea el archivo a memoria
		gommap.PROT_READ|gommap.PROT_WRITE, // Permisos de lectura y escritura
		gommap.MAP_SHARED,                  // Mapeo compartido
	); err != nil {
		return nil, err // Retorna error si falla
	}
	return idx, nil // Retorna la instancia de Index
}

// Write escribe un offset y una posición en el índice.
func (i *Index) Write(off uint32, pos uint64) error {
	if uint64(len(i.mmap)) < i.size+entWidth { // Verifica si hay espacio suficiente en el mapeo
		return io.EOF // Retorna error si no hay espacio
	}
	enc.PutUint32(i.mmap[i.size:i.size+offWidth], off)          // Escribe el offset en el mapeo
	enc.PutUint64(i.mmap[i.size+offWidth:i.size+entWidth], pos) // Escribe la posición en el mapeo
	i.size += uint64(entWidth)                                  // Incrementa el tamaño del índice
	return nil                                                  // Retorna nil si no hay errores
}

// Lee el índice y retorna el offset y la posición en el archivo.
func (i *Index) Read(in int64) (out uint32, pos uint64, err error) {
	if i.size == 0 { // Verifica si el índice está vacío
		return 0, 0, io.EOF // Retorna error si está vacío
	}
	if in == -1 { // Si el número de entrada es -1
		out = uint32((i.size / entWidth) - 1) // Lee la última entrada
	} else {
		out = uint32(in) // Lee la entrada especificada
	}
	pos = uint64(out) * entWidth // Calcula la posición en el mapeo
	if i.size < pos+entWidth {   // Verifica si la posición está fuera de rango
		return 0, 0, io.EOF // Retorna error si está fuera de rango
	}
	out = enc.Uint32(i.mmap[pos : pos+offWidth])          // Lee el offset desde el mapeo
	pos = enc.Uint64(i.mmap[pos+offWidth : pos+entWidth]) // Lee la posición desde el mapeo
	return out, pos, nil                                  // Retorna el offset y la posición
}

// Close cierra el archivo del índice, asegurando que todos los cambios se escriban en el disco.
func (i *Index) Close() error {
	if err := i.mmap.Sync(gommap.MS_SYNC); err != nil { // Sincroniza el mapeo con el disco
		return err // Retorna error si falla
	}
	if err := i.file.Sync(); err != nil { // Sincroniza el archivo con el disco
		return err // Retorna error si falla
	}
	if err := i.file.Truncate(int64(i.size)); err != nil { // Trunca el archivo al tamaño actual del índice
		return err // Retorna error si falla
	}
	return i.file.Close() // Cierra el archivo y retorna nil si no hay errores
}

// Name devuelve el nombre del archivo asociado con el índice.
func (i *Index) Name() string {
	return i.file.Name() // Retorna el nombre del archivo
}
