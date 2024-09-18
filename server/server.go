package server

import (
	"context"

	api "github.com/dati/api/v1"

	log "github.com/dati/log"
)

// Verifica en tiempo de compilación que grpcServer implemente la interfaz api.LogServer
var _ api.LogServer = (*grpcServer)(nil)

// Define la estructura grpcServer
type grpcServer struct {
	api.UnimplementedLogServer          // Embebe la implementación del servidor de log
	CommitLog                  *log.Log // Referencia al log de commits
}

// NewGRPCServer crea una nueva instancia de grpcServer
func NewGRPCServer(commitlog *log.Log) (srv *grpcServer, err error) {
	srv = &grpcServer{
		CommitLog: commitlog, // Asigna el log de commits
	}
	return srv, nil // Retorna la nueva instancia de grpcServer
}

// Produce maneja la solicitud de producción de registros
func (s *grpcServer) Produce(ctx context.Context, req *api.ProduceRequest) (*api.ProduceResponse, error) {
	offset, err := s.CommitLog.Append(req.Record) // Agrega el registro al log de commits
	if err != nil {
		return nil, err // Retorna error si ocurre
	}
	return &api.ProduceResponse{Offset: offset}, nil // Retorna la respuesta con el offset del registro
}

// Consume maneja la solicitud de consumo de registros
func (s *grpcServer) Consume(ctx context.Context, req *api.ConsumeRequest) (*api.ConsumeResponse, error) {
	record, err := s.CommitLog.Read(req.Offset) // Lee el registro del log de commits
	if err != nil {
		re, ok := err.(*api.ErrOffsetOutOfRange) // Verifica si el error es de tipo ErrOffsetOutOfRange
		if ok {
			return nil, re.GRPCStatus().Err() // Retorna el error de estado gRPC
		}
		return nil, err // Retorna otro tipo de error
	}
	return &api.ConsumeResponse{Record: record}, nil // Retorna la respuesta con el registro
}

// ProduceStream maneja la transmisión de producción de registros
func (s *grpcServer) ProduceStream(stream api.Log_ProduceStreamServer) error {
	for {
		req, err := stream.Recv() // Recibe una solicitud del stream
		if err != nil {
			return err // Retorna error si ocurre
		}
		res, err := s.Produce(stream.Context(), req) // Produce el registro
		if err != nil {
			return err // Retorna error si ocurre
		}
		if err = stream.Send(res); err != nil { // Envía la respuesta al cliente
			return err // Retorna error si ocurre
		}
	}
}

// ConsumeStream maneja la transmisión de consumo de registros
func (s *grpcServer) ConsumeStream(req *api.ConsumeRequest, stream api.Log_ConsumeStreamServer) error {
	for {
		select {
		case <-stream.Context().Done(): // Verifica si el contexto del stream ha terminado
			return nil // Retorna nil si el contexto ha terminado
		default:
			res, err := s.Consume(stream.Context(), req) // Consume el registro
			switch err.(type) {
			case nil:
			case api.ErrOffsetOutOfRange: // Maneja el error de offset fuera de rango
				continue // Continúa con la siguiente iteración
			default:
				return err // Retorna otro tipo de error
			}
			if err = stream.Send(res); err != nil { // Envía la respuesta al cliente
				return err // Retorna error si ocurre
			}
			req.Offset++ // Incrementa el offset para el siguiente registro
		}
	}
}
