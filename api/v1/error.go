package v1

import (
	"fmt"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

// Código proporcionado de los status de error para el api

type ErrOffsetOutOfRange struct {
	Offset uint64
}

func (e ErrOffsetOutOfRange) GRPCStatus() *status.Status {
	st := status.New(
		404,
		fmt.Sprintf("offset fuera de rango: %d", e.Offset),
	)
	msg := fmt.Sprintf(
		"El offset solicitado está fuera del rango del log: %d",
		e.Offset,
	)
	d := &errdetails.LocalizedMessage{
		Locale:  "es-MX",
		Message: msg,
	}
	std, err := st.WithDetails(d)
	if err != nil {
		return st
	}
	return std
}

func (e ErrOffsetOutOfRange) Error() string {
	return e.GRPCStatus().Err().Error()
}
