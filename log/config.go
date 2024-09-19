package log

// Config es la estructura que contiene configuraciones específicas para el índice,
// incluyendo el tamaño máximo permitido para el store y el índice.
type Config struct {
	Segment struct {
		MaxStoreBytes uint64 // Tamaño máximo permitido para el store
		MaxIndexBytes uint64 // Tamaño máximo permitido para el índice
		InitialOffset uint64 // Offset inicial
	}
}
