# GoLang HTTP Server

Este proyecto implementa un servidor HTTP para la clase de Cómputo Distribuido.

## Características

- **Persistencia**: Los registros se almacenan en el disco para garantizar que los datos no se pierdan, incluso en caso de fallos del sistema.
- **Segmentación**: Los registros se dividen en segmentos, lo que facilita la gestión de grandes volúmenes de datos.
- **Índice Rápido**: Un índice mapea los offsets de los registros a sus posiciones en el archivo de datos, permitiendo un acceso eficiente.
- **API HTTP**: Exposición de una API RESTful que permite agregar y recuperar registros.

## Estructura del Proyecto

- **`log/`**: Contiene la implementación principal del sistema de logging.
  - **`store.go`**: Maneja la escritura y lectura de registros en el archivo de datos.
  - **`index.go`**: Mapea los offsets de los registros a sus posiciones en el archivo de datos.
  - **`segment.go`**: Agrupa un archivo de datos y un índice en un segmento.
  - **`log.go`**: Este maneja la lógica general del log, como la gestión de múltiples segmentos y la coordinación entre `store`, `index`, y `segment`.
- **`api/`**: Proporciona una API HTTP para interactuar con el sistema de logging.
  - **`server.go`**: Configura y ejecuta el servidor HTTP, manejando las rutas para agregar y recuperar registros.
- **`main.go`**: Punto de entrada para ejecutar el servidor.
