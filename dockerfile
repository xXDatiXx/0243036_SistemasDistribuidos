# Usa una imagen base de Go para la compilación
FROM golang:1.23 AS builder

# Establece el directorio de trabajo dentro del contenedor
WORKDIR /app

# Copia los archivos go.mod y go.sum y descarga las dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copia todos los archivos del proyecto al contenedor
COPY . .

# Compila la aplicación principal
RUN go build -o main ./server/server.go

# Ejecuta los tests (opcional: puedes ejecutar `go test ./server` para diagnosticar el error)
RUN go test -v ./server

# Usa una imagen más ligera para el runtime
FROM alpine:latest

# Copia el binario de la aplicación desde la etapa builder
COPY --from=builder /app/main /app/main

# Exponer el puerto en el que correrá la aplicación principal
EXPOSE 8080

# Comando por defecto para iniciar la aplicación
CMD ["/app/main"]
