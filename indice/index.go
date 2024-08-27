package index

import (
    "encoding/binary"
    "os"
    "sync"
    "unsafe"

    "github.com/tysonmote/gommap"
)

const (
    entWidth = 12
    offWidth = 4
)

type Index struct {
    file *os.File
    mmap gommap.MMap
    size uint64
    mu   sync.Mutex
}

// Función para crear un nuevo índice a partir de un archivo
func NewIndex(f *os.File) (*Index, error) {
    fi, err := f.Stat()
    if err != nil {
        return nil, err
    }

    size := uint64(fi.Size())
    if err := f.Truncate(int64(size)); err != nil {
        return nil, err
    }

    // Mapear el archivo a memoria
    mmap, err := gommap.MapRegion(
        f.Fd(),
        int64(size),
        gommap.PROT_READ|gommap.PROT_WRITE,
        gommap.MAP_SHARED,
        0,
    )
    if err != nil {
        return nil, err
    }

    return &Index{
        file: f,
        mmap: mmap,
        size: size,
    }, nil
}

// Función para leer un registro desde el índice
func (i *Index) Read(in int64) (out uint32, pos uint64, err error) {
    i.mu.Lock()
    defer i.mu.Unlock()

    if i.size == 0 {
        return 0, 0, os.ErrNotExist
    }

    out = binary.BigEndian.Uint32(i.mmap[in*int64(entWidth):])
    pos = binary.BigEndian.Uint64(i.mmap[in*int64(entWidth)+int64(offWidth):])
    return out, pos, nil
}

// Función para escribir un registro en el índice
func (i *Index) Write(off uint32, pos uint64) error {
    i.mu.Lock()
    defer i.mu.Unlock()

    if uint64(len(i.mmap)) < i.size+entWidth {
        return os.ErrInvalid
    }

    binary.BigEndian.PutUint32(i.mmap[i.size:], off)
    binary.BigEndian.PutUint64(i.mmap[i.size+offWidth:], pos)
    i.size += entWidth

    return nil
}

// Función para cerrar el índice
func (i *Index) Close() error {
    i.mu.Lock()
    defer i.mu.Unlock()

    if err := i.mmap.Sync(gommap.MS_SYNC); err != nil {
        return err
    }

    if err := i.file.Sync(); err != nil {
        return err
    }

    if err := i.file.Truncate(int64(i.size)); err != nil {
        return err
    }

    return i.file.Close()
}