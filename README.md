# GoLang Distributed Logging System

Este proyecto implementa un sistema de logging distribuido con almacenamiento persistente, segmentación y gRPC para la clase de Cómputo Distribuido.

## Características

- **Persistencia**: Los registros se almacenan en el disco para garantizar que los datos no se pierdan, incluso en caso de fallos del sistema.
- **Segmentación**: Los registros se dividen en segmentos, lo que facilita la gestión de grandes volúmenes de datos y permite un acceso más eficiente.
- **Índice Rápido**: Un índice mapea los offsets de los registros a sus posiciones en el archivo de datos, permitiendo un acceso eficiente y rápido a los registros almacenados.
- ** gRPC**: Provee gRPC para la producción y consumo de registros, permitiendo interacciones más eficientes y robustas en un entorno distribuido.
- **Soporte para Streams**: Permite la producción y el consumo de registros a través de streams gRPC, facilitando la transmisión continua de datos.

## Estructura del Proyecto

- **`log/`**: Contiene la implementación principal del sistema de logging.
  - **`store.go`**: Maneja la escritura y lectura de registros en el archivo de datos. Utiliza un buffer para optimizar la escritura en disco y asegura el acceso concurrente mediante el uso de mutex.
  - **`index.go`**: Implementa un índice para mapear los offsets de los registros a sus posiciones en el archivo de datos. Facilita la búsqueda rápida y eficiente de registros.
  - **`segment.go`**: Agrupa un archivo de datos (`store`) y un índice (`index`) en un segmento. Cada segmento tiene un tamaño máximo configurable, y se crean nuevos segmentos cuando el actual alcanza su límite.
  - **`log.go`**: Implementa la lógica general del log, gestionando múltiples segmentos y coordinando las operaciones de almacenamiento (`store`), indexación (`index`), y segmentación (`segment`).
- **`api/v1/`**: Contiene la definición de la API gRPC y la implementación de los servicios.
  - **`log.proto`**: Define la estructura de los mensajes y servicios gRPC para la interacción con el sistema de logging.
- **`server/`**: Contiene la implementación de server.go.
  - **`server.go`**: Implementa el servidor gRPC, proporcionando métodos para producir y consumir registros, así como para manejar streams de datos.

## Flujo del Proyecto

### 1. **Inicialización del Log**
   - Cuando se crea una instancia de `Log`, se leen todos los segmentos existentes desde el directorio especificado.
   - Si no hay segmentos, se crea un nuevo segmento con un `offset` inicial definido en la configuración.

### 2. **Almacenamiento de Registros (Produce)**
   - Los registros se agregan a través del método `Append` de la estructura `Log`.
   - El registro se serializa y se almacena en el archivo correspondiente del `store`.
   - Se actualiza el `index` con el nuevo `offset` y la posición del registro en el archivo `store`.
   - Si el segmento actual alcanza su tamaño máximo (`MaxStoreBytes` o `MaxIndexBytes`), se crea un nuevo segmento para continuar almacenando registros.

### 3. **Lectura de Registros (Consume)**
   - Para leer un registro específico, se proporciona su `offset` al método `Read` del `Log`.
   - Se busca el segmento correspondiente que contiene el `offset`.
   - Se utiliza el `index` del segmento para encontrar la posición del registro en el `store`.
   - Se lee y se deserializa el registro desde el `store` y se devuelve al cliente.

### 4. **gRPC**
   - El servidor gRPC expone métodos como `Produce` y `Consume` para agregar y leer registros, respectivamente.
   - También soporta `ProduceStream` y `ConsumeStream` para la transmisión continua de datos, útil en aplicaciones donde se requiere un flujo constante de información.

### 5. **Manejo de Errores**
   - Si se intenta leer un registro con un `offset` fuera del rango, se retorna un error de tipo `ErrOffsetOutOfRange`.

### 6. **Cierre y Eliminación de Segmentos**
   - Los segmentos pueden cerrarse para liberar recursos. Esto asegura que todos los datos se escriban en disco y se cierre el acceso a los archivos.
   - Los segmentos obsoletos o innecesarios se pueden eliminar para liberar espacio en disco, manteniendo solo los registros recientes o relevantes.