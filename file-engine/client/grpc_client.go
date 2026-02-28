package client

import (
	"context"
	"fmt"

	pb "github.com/example/file-engine/pkg/generated"
	"google.golang.org/grpc"
)

type GRPCClient struct {
	conn   *grpc.ClientConn
	client pb.FileEngineClient
}

func NewGRPCClient(addr string, opts ...grpc.DialOption) (*GRPCClient, error) {
	if len(opts) == 0 {
		return nil, fmt.Errorf("missing grpc.DialOption: explicit transport credentials required")
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("create grpc client: %w", err)
	}
	return &GRPCClient{conn: conn, client: pb.NewFileEngineClient(conn)}, nil
}

func (c *GRPCClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *GRPCClient) CreateFolder(ctx context.Context, req *pb.CreateFolderRequest) (*pb.CreateFolderResponse, error) {
	resp, err := c.client.CreateFolder(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("CreateFolder: %w", err)
	}
	return resp, nil
}

func (c *GRPCClient) GetTaskStatus(ctx context.Context, req *pb.TaskStatusRequest) (*pb.TaskStatusResponse, error) {
	resp, err := c.client.GetTaskStatus(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("GetTaskStatus: %w", err)
	}
	return resp, nil
}
