package client

import (
	"context"
	"fmt"

	pb "github.com/example/file-engine/pkg/generated"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCClient struct {
	conn   *grpc.ClientConn
	client pb.FileEngineClient
}

func NewGRPCClient(addr string, opts ...grpc.DialOption) (*GRPCClient, error) {
	if len(opts) == 0 {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("create grpc client: %w", err)
	}
	return &GRPCClient{conn: conn, client: pb.NewFileEngineClient(conn)}, nil
}

func (c *GRPCClient) Close() error { return c.conn.Close() }

func (c *GRPCClient) CreateFolder(ctx context.Context, req *pb.CreateFolderRequest) (*pb.CreateFolderResponse, error) {
	return c.client.CreateFolder(ctx, req)
}

func (c *GRPCClient) GetTaskStatus(ctx context.Context, req *pb.TaskStatusRequest) (*pb.TaskStatusResponse, error) {
	return c.client.GetTaskStatus(ctx, req)
}
