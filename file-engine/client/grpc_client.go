package client

import (
	"context"

	pb "github.com/example/file-engine/pkg/generated"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCClient struct {
	conn   *grpc.ClientConn
	client pb.FileEngineClient
}

// NewGRPCClient creates a GRPCClient connected to the given address using insecure transport credentials.
// It returns a GRPCClient that wraps the established gRPC connection and the generated FileEngineClient, or an error if the connection cannot be established.
func NewGRPCClient(addr string) (*GRPCClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
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
