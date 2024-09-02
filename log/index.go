package log

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/tysonmote/gommap"
)

// Variables que definen el ancho de los campos en el índice.
// offWidth es el tamaño del offset (4 bytes), posWidth es el tamaño de la posición (8 bytes),
// y entWidth es el tamaño total de una entrada en el índice.
var (
	offWidth uint64 = 4
	posWidth uint64 = 8
	entWidth        = int(offWidth + posWidth) // Tamaño total de una entrada en el índice
)

// Index representa el índice de un segmento, que mapea offsets a posiciones en el store.
type Index struct {
	file *os.File    // Archivo en el cual se almacena el índice
	mmap gommap.MMap // Mapeo de memoria para acceder al archivo del índice
	size uint64      // Tamaño actual del índice en bytes
	mu   sync.Mutex  // Mutex para asegurar acceso concurrente seguro
}

// Config es la estructura que contiene configuraciones específicas para el índice,
// incluyendo el tamaño máximo permitido para el store y el índice.
type Config struct {
	Segment struct {
		MaxStoreBytes uint64 // Tamaño máximo permitido para el store en bytes
		MaxIndexBytes uint64 // Tamaño máximo permitido para el índice en bytes
		InitialOffset uint64 // Offset inicial del segmento
	}
}

// NewIndex crea un nuevo índice a partir de un archivo dado y configura el mapeo a memoria.
// Devuelve una instancia de Index o un error si falla.
func NewIndex(f *os.File, c Config) (*Index, error) {
	// 1. Obtener el tamaño del archivo que vamos a indexar.
	fi, err := f.Stat()
	if err != nil {
		fmt.Printf("Error al obtener el tamaño del archivo: %v\n", err)
		return nil, err
	}

	// Inicializa una nueva instancia de Index con el archivo y su tamaño.
	idx := &Index{
		file: f,
		size: uint64(fi.Size()),
	}

	// Ajusta el tamaño del archivo de índice al tamaño máximo configurado.
	if err := os.Truncate(f.Name(), int64(c.Segment.MaxIndexBytes)); err != nil {
		fmt.Printf("Error al truncar el archivo: %v\n", err)
		return nil, err
	}

	// Mapea el archivo directamente a memoria utilizando gommap.
	if idx.mmap, err = gommap.Map(idx.file.Fd(), gommap.PROT_READ|gommap.PROT_WRITE, gommap.MAP_SHARED); err != nil {
		fmt.Printf("Error al mapear el archivo a memoria: %v\n", err)
		return nil, err
	}

	// Si el tamaño es 0, limpiar el mmap para asegurar que no haya datos basura.
	if len(idx.mmap) > 0 && idx.size == 0 {
		for i := range idx.mmap {
			idx.mmap[i] = 0
		}
	}

	return idx, nil
}

// Write escribe un offset y una posición en el índice.
func (i *Index) Write(off uint32, pos uint64) error {
	i.mu.Lock() // Bloquear para acceso seguro.
	defer i.mu.Unlock()

	// Verifica si hay suficiente espacio para escribir una nueva entrada.
	if i.size+uint64(entWidth) > uint64(len(i.mmap)) {
		fmt.Println("Error: No hay suficiente espacio en el índice para escribir una nueva entrada.")
		return io.EOF
	}

	// Escribe el offset en el índice.
	binary.BigEndian.PutUint32(i.mmap[i.size:], off)

	// Escribir la posición asociada al offset en el índice.
	binary.BigEndian.PutUint64(i.mmap[i.size+uint64(offWidth):], pos)

	// Incrementar el tamaño del índice para reflejar la nueva entrada.
	i.size += uint64(entWidth)

	return nil
}

// Read lee una entrada desde el índice basado en el número de entrada especificado.
func (i *Index) Read(in int64) (out uint32, pos uint64, err error) {
	i.mu.Lock() // Bloquear para acceso seguro.
	defer i.mu.Unlock()

	// Verificar si el índice contiene registros.
	if i.size == 0 {
		fmt.Println("Error: No hay registros en el índice.")
		return 0, 0, io.EOF
	}

	// Si in es -1, que significa que la última entrada en el índice.
	if in == -1 {
		out = uint32((i.size / uint64(entWidth)) - 1)
		offsetPos := i.size - uint64(entWidth)

		// Leer la posición desde la última entrada en el índice.
		pos = binary.BigEndian.Uint64(i.mmap[offsetPos+uint64(offWidth):])

		return out, pos, nil
	}

	// Calcular la posición de la entrada solicitada.
	entryOffset := uint64(in) * uint64(entWidth)
	if in < 0 || entryOffset >= i.size {
		fmt.Println("Error: Entrada fuera de los límites del índice.")
		return 0, 0, io.EOF
	}

	// Leer el offset y la posición desde el índice.
	out = binary.BigEndian.Uint32(i.mmap[entryOffset:])
	pos = binary.BigEndian.Uint64(i.mmap[entryOffset+uint64(offWidth):])

	return out, pos, nil
}

// Close cierra el archivo del índice, asegurando que todos los cambios se escriban en el disco.
func (i *Index) Close() error {
	i.mu.Lock() // Bloquear para acceso seguro.
	defer i.mu.Unlock()

	// Hace flush de los datos en memoria al archivo.
	if err := i.mmap.Sync(gommap.MS_SYNC); err != nil {
		fmt.Printf("Error al sincronizar el mmap con el archivo: %v\n", err)
		return err
	}

	// Sincroniza el archivo para que todos los datos se escriban en el disco.
	if err := i.file.Sync(); err != nil {
		fmt.Printf("Error al sincronizar el archivo: %v\n", err)
		return err
	}

	// Limitar el archivo para que tenga el tamaño exacto de su contenido.
	if err := i.file.Truncate(int64(i.size)); err != nil {
		fmt.Printf("Error al truncar el archivo: %v\n", err)
		return err
	}

	// Cierra el archivo.
	if err := i.file.Close(); err != nil {
		fmt.Printf("Error al cerrar el archivo: %v\n", err)
		return err
	}

	return nil
}

// Name devuelve el nombre del archivo asociado con el índice.
func (i *Index) Name() string {
	return i.file.Name()
}
