package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"

	api "github.com/dati/api/v1" // Cambien esto por la ruta en su máquina
	"github.com/dati/log"        // Cambien esto por la ruta en su máquina

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
		client api.LogClient,
		config *Config,
	){
		"produce/consume a message to/from the log succeeeds": testProduceConsume,
		"produce/consume stream succeeds":                     testProduceConsumeStream,
		"consume past log boundary fails":                     testConsumePastBoundary,
	} {
		t.Run(scenario, func(t *testing.T) {
			client, config, teardown := setupTest(t, nil)
			defer teardown()
			fn(t, client, config)
		})
	}
}

// END: intro

// START: setup
func setupTest(t *testing.T, fn func(*Config)) (
	client api.LogClient,
	config *Config,
	teardown func(),
) {
	t.Helper()

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	clientOptions := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	cc, err := grpc.NewClient(l.Addr().String(), clientOptions...)
	require.NoError(t, err)

	dir, err := os.MkdirTemp("", "server-test")
	require.NoError(t, err)

	clog, err := log.NewLog(dir, log.Config{})
	require.NoError(t, err)

	config = &Config{
		CommitLog: clog,
	}
	if fn != nil {
		fn(config)
	}
	server, err := NewGRPCServer(config)
	require.NoError(t, err)

	go func() {
		server.Serve(l)
	}()

	client = api.NewLogClient(cc)

	return client, config, func() {
		server.Stop()
		cc.Close()
		l.Close()
		clog.Remove()
	}
}

// END: setup

// START: produceconsume
func testProduceConsume(t *testing.T, client api.LogClient, config *Config) {
	ctx := context.Background()

	want := &api.Record{
		Value: []byte("hello world"),
	}

	produce, err := client.Produce(
		ctx,
		&api.ProduceRequest{
			Record: want,
		},
	)
	require.NoError(t, err)

	consume, err := client.Consume(ctx, &api.ConsumeRequest{
		Offset: produce.Offset,
	})
	require.NoError(t, err)
	require.Equal(t, want.Value, consume.Record.Value)
	require.Equal(t, want.Offset, consume.Record.Offset)
}

// END: produceconsume

// START: consumeerror
func testConsumePastBoundary(
	t *testing.T,
	client api.LogClient,
	config *Config,
) {
	ctx := context.Background()

	produce, err := client.Produce(ctx, &api.ProduceRequest{
		Record: &api.Record{
			Value: []byte("hello world"),
		},
	})
	require.NoError(t, err)

	consume, err := client.Consume(ctx, &api.ConsumeRequest{
		Offset: produce.Offset + 1,
	})
	if consume != nil {
		t.Fatal("consume not nil")
	}
	fmt.Printf("Error recibido: %v\n", err)
	got := status.Code(err)
	want := status.Code(api.ErrOffsetOutOfRange{}.GRPCStatus().Err())
	if got != want {
		t.Fatalf("got err: %v, want: %v", got, want)
	}
}

// END: consumeerror

// START: stream
func testProduceConsumeStream(
	t *testing.T,
	client api.LogClient,
	config *Config,
) {
	ctx := context.Background()

	records := []*api.Record{{
		Value:  []byte("first message"),
		Offset: 0,
	}, {
		Value:  []byte("second message"),
		Offset: 1,
	}}

	{
		stream, err := client.ProduceStream(ctx)
		require.NoError(t, err)

		for offset, record := range records {
			err = stream.Send(&api.ProduceRequest{
				Record: record,
			})
			require.NoError(t, err)
			res, err := stream.Recv()
			require.NoError(t, err)
			if res.Offset != uint64(offset) {
				t.Fatalf(
					"got offset: %d, want: %d",
					res.Offset,
					offset,
				)
			}
		}

	}

	{
		stream, err := client.ConsumeStream(
			ctx,
			&api.ConsumeRequest{Offset: 0},
		)
		require.NoError(t, err)

		for i, record := range records {
			res, err := stream.Recv()
			require.NoError(t, err)
			require.Equal(t, res.Record, &api.Record{
				Value:  record.Value,
				Offset: uint64(i),
			})
		}
	}
}
