package index

import (
	"encoding/binary"
	"fmt"
	"os"
	"syscall"

	"github.com/tysonmote/gommap"
)

var (
	offWidth    = uint64(4)           // Tamaño en bytes del offset
	posWidth    = uint64(8)           // Tamaño en bytes de la posición
	entWidth    = offWidth + posWidth // Tamaño total de una entrada
	initialSize = uint64(1024)        // Tamaño inicial predeterminado si el archivo está vacío
)

// Estructura del índice
type Index struct {
	file *os.File    // Archivo donde se almacena el índice
	mmap gommap.MMap // Mapeo del archivo a memoria
	size uint64      // Tamaño actual utilizado del índice
}

// NewIndex crea un nuevo índice basado en un archivo
func NewIndex(f *os.File) (*Index, error) {
	// 1. Obtener el tamaño del archivo que vamos a indexar
	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("error al obtener el tamaño del archivo: %v", err)
	}

	size := uint64(fi.Size())
	fmt.Printf("Tamaño del archivo: %d bytes\n", size)

	// 2. Si el archivo está vacío, inicializarlo con un tamaño predeterminado
	if size == 0 {
		fmt.Println("El archivo de índice está vacío. Inicializando con tamaño predeterminado...")
		size = initialSize
		if err := f.Truncate(int64(size)); err != nil {
			return nil, fmt.Errorf("error al inicializar el archivo de índice: %v", err)
		}
	}

	// 3. Hacer el mapeo directo entre archivo y memoria usando syscall.Mmap
	mmap, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("error al mapear el archivo a memoria: %v", err)
	}

	// Devolver la estructura del índice
	return &Index{
		file: f,
		mmap: mmap,
		size: size,
	}, nil
}

// Función para leer un registro desde el índice
func (i *Index) Read(in int64) (out uint32, pos uint64, err error) {
	// 1. Obtener el lugar en donde queremos leer
	entryStart := uint64(in) * entWidth

	// Verificación de que estamos dentro del rango del tamaño del índice
	if entryStart+entWidth > i.size {
		return 0, 0, fmt.Errorf("entrada fuera de los límites del índice")
	}

	// 2. Decodificación binaria para obtener el offset y la posición del Store
	// 2.1 Decodificar el offset
	out = binary.BigEndian.Uint32(i.mmap[entryStart : entryStart+offWidth])

	// 2.2 Decodificar la posición del Store
	pos = binary.BigEndian.Uint64(i.mmap[entryStart+offWidth : entryStart+entWidth])

	// Devolver el offset y la posición del Store
	return out, pos, nil
}

// Función para escribir un registro en el índice
func (i *Index) Write(off uint32, pos uint64) error {
	// 1. Obtener el tamaño actual del índice y validar que tenemos espacio para escribir
	if i.size+entWidth > uint64(len(i.mmap)) {
		return fmt.Errorf("no hay suficiente espacio en el índice para escribir una nueva entrada")
	}

	// 2. Escribir el offset en los primeros bits del espacio disponible
	binary.BigEndian.PutUint32(i.mmap[i.size:i.size+offWidth], off)

	// 3. Escribir la posición desde el final del offset hasta el final de la posición
	binary.BigEndian.PutUint64(i.mmap[i.size+offWidth:i.size+entWidth], pos)

	// Actualizar el tamaño del índice después de la escritura
	i.size += entWidth

	return nil
}

// Función para cerrar el índice y realizar las operaciones necesarias antes del cierre
func (i *Index) Close() error {
	// 1. Hacer flush de los datos en memoria a los datos en el archivo
	if err := i.mmap.Sync(gommap.MS_SYNC); err != nil {
		return fmt.Errorf("error al sincronizar el mmap con el archivo: %w", err)
	}

	// 2. Truncar el archivo para que tenga el tamaño justo de su contenido
	if err := i.file.Truncate(int64(i.size)); err != nil {
		return fmt.Errorf("error al truncar el archivo: %w", err)
	}

	// 3. Desmapear el archivo de la memoria
	if err := i.mmap.UnsafeUnmap(); err != nil {
		return fmt.Errorf("error al desmapear el archivo: %w", err)
	}

	// 4. Cerrar el archivo
	if err := i.file.Close(); err != nil {
		return fmt.Errorf("error al cerrar el archivo: %w", err)
	}

	return nil
}
