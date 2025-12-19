package server

import (
    "fmt"
)

type GRPCServer struct {
    Addr string
}

type HTTPServer struct {
    Addr string
    ProxyTo string
}

func NewGRPCServer(addr string) *GRPCServer {
    return &GRPCServer{Addr: addr}
}
func NewHTTPServer(addr, proxy string) *HTTPServer {
    return &HTTPServer{Addr: addr, ProxyTo: proxy}
}

func (g *GRPCServer) Start() error {
    fmt.Println("Starting gRPC server on", g.Addr)
    select {}
    return nil
}

func (h *HTTPServer) Start() error {
    fmt.Println("Starting HTTP gateway on", h.Addr, "proxy->", h.ProxyTo)
    select {}
    return nil
}