package common

import (
	"context"
	"im/pkg/config"
	apigatewayService "im/server/apigateway/rpc/service"
	imGatewayService "im/server/imgateway/rpc/service"
	"log/slog"
	"net"

	"fyne.io/fyne/v2"
)

type Context struct {
	App               fyne.App
	Ctx               context.Context
	Logger            *slog.Logger
	Config            *config.ClientConfig
	ApiGatewayClient  apigatewayService.APIGatewayClient
	IMGatewayClient   imGatewayService.IMGatewayClient
	IMGatewayLongConn net.Conn
	MessageReadChan   chan ChatMessage
	MessageWriteChan  chan ChatMessage
	LoginPage         fyne.Window
	HomePage          fyne.Window
	Token             string
	RefreshToken      string
	User              *User
	SessionUserTable  map[string]map[string]User // TODO 并发安全
}

type User struct {
	UUID   string
	Name   string
	Avatar string
	Email  string
	Phone  string
}
