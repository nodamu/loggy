package server

import (
	"context"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	logv1 "loggy/api/v1"
)

type CommitLog interface {
	Append(*logv1.Record) (uint64, error)
	Read(uint64) (*logv1.Record, error)
}

type Config struct {
	CommitLog  CommitLog
	Authorizer Authorizer
}

const (
	objectWildCard = "*"
	produceAction  = "produce"
	consumeAction  = "consume"
)

var _ logv1.LogServer = (*grpcServer)(nil)

type Authorizer interface {
	Authorize(subject, object, action string) error
}

type grpcServer struct {
	logv1.UnimplementedLogServer
	*Config
}

func NewGRPCServer(config *Config, opts ...grpc.ServerOption) (*grpc.Server, error) {
	opts = append(
		opts,
		grpc.ChainStreamInterceptor(grpc_auth.StreamServerInterceptor(authenticate)),
		grpc.ChainUnaryInterceptor(grpc_auth.UnaryServerInterceptor(authenticate)),
	)
	gsrv := grpc.NewServer(opts...)
	srv, err := newGRPCServer(config)
	if err != nil {
		return nil, err
	}
	logv1.RegisterLogServer(gsrv, srv)
	return gsrv, nil
}

func newGRPCServer(config *Config) (srv *grpcServer, err error) {
	srv = &grpcServer{
		Config: config,
	}
	return srv, nil
}

type subjectContextKey struct{}

func subject(ctx context.Context) string {
	return ctx.Value(subjectContextKey{}).(string)
}

func authenticate(ctx context.Context) (context.Context, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, status.New(codes.Unknown, "could not find peer info").Err()
	}
	if peer.AuthInfo == nil {
		return context.WithValue(ctx, subjectContextKey{}, ""), nil
	}
	tlsInfo := peer.AuthInfo.(credentials.TLSInfo)
	subject := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	ctx = context.WithValue(ctx, subjectContextKey{}, subject)
	return ctx, nil
}

func (s *grpcServer) Produce(
	ctx context.Context,
	req *logv1.ProduceRequest,
) (*logv1.ProduceResponse, error) {
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildCard,
		produceAction,
	); err != nil {
		return nil, err
	}
	offSet, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}
	return &logv1.ProduceResponse{Offset: offSet}, nil
}

func (s *grpcServer) Consume(
	ctx context.Context,
	req *logv1.ConsumeRequest,
) (*logv1.ConsumeResponse, error) {
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildCard,
		consumeAction,
	); err != nil {
		return nil, err
	}

	r, err := s.CommitLog.Read(req.Offset)
	if err != nil {
		return nil, err
	}
	return &logv1.ConsumeResponse{Record: r}, nil
}

func (s *grpcServer) ProduceStream(stream logv1.Log_ProduceStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}
		res, err := s.Produce(stream.Context(), req)
		if err != nil {
			return err
		}
		if err := stream.Send(res); err != nil {
			return err
		}

	}
}

func (s *grpcServer) ConsumeStream(
	req *logv1.ConsumeRequest,
	stream logv1.Log_ConsumeStreamServer,
) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			res, err := s.Consume(stream.Context(), req)
			switch err.(type) {
			case nil:
			case logv1.ErrOffsetOutOfRange:
				continue
			default:
				return err
			}
			if err := stream.Send(res); err != nil {
				return err
			}
			req.Offset++
		}
	}

}
