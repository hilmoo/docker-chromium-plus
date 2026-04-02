package main

import (
	"context"
	"log/slog"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newStdio(ctx context.Context, cfg config) (*mcp.ClientSession, error) {
	browserURL := "http://localhost:" + cfg.ChromePort
	cmd := exec.CommandContext(ctx, "chrome-devtools-mcp", "--browser-url="+browserURL, "--no-usage-statistics")
	slog.Info("Starting stdio process", "command", cmd.String())

	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "chrome-devtools-mcp-client",
		Version: MCPVersion,
	}, nil)

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		slog.Error("Failed to connect to stdio server", "error", err)
		return nil, err
	}

	return session, nil
}

func newServer(session *mcp.ClientSession, logger *slog.Logger) *mcp.Server {
	proxyServer := mcp.NewServer(&mcp.Implementation{
		Name:    "chrome-devtools-mcp-proxy",
		Version: MCPVersion,
	}, &mcp.ServerOptions{
		Logger:   logger,
		HasTools: true,
	})

	proxyServer.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			switch method {
			case "tools/list":
				return session.ListTools(ctx, nil)
			case "tools/call":
				if callReq, ok := req.(*mcp.CallToolRequest); ok {
					return session.CallTool(ctx, &mcp.CallToolParams{
						Name:      callReq.Params.Name,
						Arguments: callReq.Params.Arguments,
					})
				}
			}
			return next(ctx, method, req)
		}
	})

	return proxyServer
}
