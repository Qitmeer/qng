package mcp

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/meerevm/meer"
	"github.com/ethereum/go-ethereum"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

func (m *MCPService) registerTools() error {
	toolhs := []ToolHelp{}
	for _, js := range ethereum.Modules {
		ths, err := ParseToolHelps(js)
		if err != nil {
			log.Error(err.Error())
			continue
		}
		toolhs = append(toolhs, ths...)
	}
	log.Info("Parse apis for MCP tool", "size", len(toolhs))

	for _, th := range toolhs {
		m.mcpServer.AddTool(mcpgo.NewTool(th.Method, mcpgo.WithDescription(fmt.Sprintf("Params num is %d.(The parameter name needs to end with a serial number, for example, parameter-1)", th.ParamsNum))), m.toolFunc)
	}
	return nil
}

func (m *MCPService) toolFunc(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if len(request.Params.Name) <= 0 {
		return nil, fmt.Errorf("No method name")
	}
	client := m.rpcSer.BC.MeerChain().(*meer.BlockChain).Client()
	if client == nil || client.Client() == nil {
		return nil, fmt.Errorf("Serve not init")
	}
	params := ValuesByNumericKeyOrder(request.Params.Arguments)
	tr := ToolResult{}
	if len(params) <= 0 {
		err := client.Client().Call(&tr, request.Params.Name)
		if err != nil {
			return nil, err
		}
	} else {
		err := client.Client().Call(&tr, request.Params.Name, params)
		if err != nil {
			return nil, err
		}
	}
	return mcpgo.NewToolResultText(string(tr.raw)), nil
}
