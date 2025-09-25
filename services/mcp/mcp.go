package mcp

import (
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/rpc"
	"github.com/mark3labs/mcp-go/server"
)

type MCPService struct {
	service.Service
	cfg       *config.Config
	mcpServer *server.MCPServer
	rpcSer    *rpc.RpcServer
}

func New(cfg *config.Config, rpcSer *rpc.RpcServer) (*MCPService, error) {
	m := &MCPService{
		cfg:    cfg,
		rpcSer: rpcSer,
	}
	err := m.initMCPServer()
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (m *MCPService) Start() error {
	log.Info("MCPService start")
	if err := m.Service.Start(); err != nil {
		return err
	}
	return nil
}

func (m *MCPService) Stop() error {
	log.Info("MCPService stop")
	if err := m.Service.Stop(); err != nil {
		return err
	}

	return nil
}

func (m *MCPService) initMCPServer() error {
	m.mcpServer = server.NewMCPServer(
		"qng-mcp-server",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithToolCapabilities(true),
	)

	err := m.registerTools()
	if err != nil {
		return err
	}

	sseServer := server.NewSSEServer(m.mcpServer, server.WithBasePath("/mcp"))
	m.rpcSer.RegisterHandler("/mcp/sse", sseServer.SSEHandler())
	m.rpcSer.RegisterHandler("/mcp/message", sseServer.MessageHandler())
	return nil
}
