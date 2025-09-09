package crm

import (
	"context"

	crmpb "github.com/YanMak/ecommerce/v2/api/gen/go/crm/certificates/v1"
	"github.com/YanMak/ecommerce/v2/pkg/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn *grpc.ClientConn
	Cert crmpb.CertificatesClient
}

func New(ctx context.Context, addr string, extra ...grpc.DialOption) (*Client, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()), // TODO: TLS позже
		grpc.WithUnaryInterceptor(grpcx.UnaryClientMetaInterceptor),
	}
	opts = append(opts, extra...)
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, Cert: crmpb.NewCertificatesClient(conn)}, nil
}

func (c *Client) Close() error { return c.conn.Close() }
