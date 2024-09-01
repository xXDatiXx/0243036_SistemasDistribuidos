package log

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/tysonmote/gommap"
)

var (
	offWidth uint64 = 4
	posWidth uint64 = 8
	entWidth        = int(offWidth + posWidth)
)

type Index struct {
	file *os.File
	mmap gommap.MMap
	size uint64
	mu   sync.Mutex
}

type Config struct {
	Segment struct {
		MaxIndexBytes int64
	}
}

// newIndex crea un nuevo índice a partir de un archivo
func NewIndex(f *os.File, c Config) (*Index, error) {
	// 1. Obtener el tamaño del archivo que vamos a Indexar
	fi, err := f.Stat()
	if err != nil {
		fmt.Printf("Error al obtener el tamaño del archivo: %v\n", err)
		return nil, err
	}

	idx := &Index{
		file: f,
		size: uint64(fi.Size()),
	}

	// 2. Establecer el tamaño del índice como el tamaño del archivo
	if err := os.Truncate(f.Name(), int64(c.Segment.MaxIndexBytes)); err != nil {
		fmt.Printf("Error al truncar el archivo: %v\n", err)
		return nil, err
	}

	// 3. Mapear el archivo directamente a memoria
	if idx.mmap, err = gommap.Map(idx.file.Fd(), gommap.PROT_READ|gommap.PROT_WRITE, gommap.MAP_SHARED); err != nil {
		fmt.Printf("Error al mapear el archivo a memoria: %v\n", err)
		return nil, err
	}

	// Si el tamaño es 0, limpiar el mmap para asegurar que no haya datos basura
	if len(idx.mmap) > 0 && idx.size == 0 {
		for i := range idx.mmap {
			idx.mmap[i] = 0
		}
	}

	return idx, nil
}

// Write escribe un offset y una posición en el índice
func (i *Index) Write(off uint32, pos uint64) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// 1. Obtener el tamaño actual del índice y validar que tenemos espacio para escribir
	if i.size+uint64(entWidth) > uint64(len(i.mmap)) {
		fmt.Println("Error: No hay suficiente espacio en el índice para escribir una nueva entrada.")
		return io.EOF
	}

	// 2. Escribir primero el offset, desde el final del archivo hasta el tamaño del offset
	binary.BigEndian.PutUint32(i.mmap[i.size:], off)

	// 3. Luego escribir la posición desde el final del offset hasta el final de la posición
	binary.BigEndian.PutUint64(i.mmap[i.size+uint64(offWidth):], pos)

	i.size += uint64(entWidth)

	return nil
}

// Read lee una entrada desde el índice
func (i *Index) Read(in int64) (out uint32, pos uint64, err error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	// 1. Obtener el lugar en donde queremos leer
	if i.size == 0 {
		fmt.Println("Error: No hay registros en el índice.")
		return 0, 0, io.EOF
	}

	if in == -1 {
		out = uint32((i.size / uint64(entWidth)) - 1)
		offsetPos := i.size - uint64(entWidth)

		// Multiplica el entero por entWidth, lo que nos da una posición inicial para decodificar desde binario
		pos = binary.BigEndian.Uint64(i.mmap[offsetPos+uint64(offWidth):])

		return out, pos, nil
	}

	entryOffset := uint64(in) * uint64(entWidth)
	if in < 0 || entryOffset >= i.size {
		fmt.Println("Error: Entrada fuera de los límites del índice.")
		return 0, 0, io.EOF
	}

	// Iniciar desde la posición que establecimos anteriormente, luego obtener desde allí hasta el final de los bytes de offset
	out = binary.BigEndian.Uint32(i.mmap[entryOffset:])

	// Para la posición, simplemente comenzar desde el final del offset hasta el entWidth, esto nos dará la posición en el Store
	pos = binary.BigEndian.Uint64(i.mmap[entryOffset+uint64(offWidth):])

	return out, pos, nil
}

// Cierra el archivo del índice
func (i *Index) Close() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// 1. Verifica que hemos hecho flush de los datos en memoria al archivo
	if err := i.mmap.Sync(gommap.MS_SYNC); err != nil {
		fmt.Printf("Error al sincronizar el mmap con el archivo: %v\n", err)
		return err
	}

	// Sincronización el archivo para que los datos se escriban
	if err := i.file.Sync(); err != nil {
		fmt.Printf("Error al sincronizar el archivo: %v\n", err)
		return err
	}

	// 2. Limita archivo para que tenga el tamaño exacto de su contenido
	if err := i.file.Truncate(int64(i.size)); err != nil {
		fmt.Printf("Error al truncar el archivo: %v\n", err)
		return err
	}

	// 3. Cerrar el archivo
	if err := i.file.Close(); err != nil {
		fmt.Printf("Error al cerrar el archivo: %v\n", err)
		return err
	}

	return nil
}

// Devuelve el nombre del archivo del índice
func (i *Index) Name() string {
	return i.file.Name()
}
