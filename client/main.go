package main

import (
	"context"
	"im/client/common"
	"im/client/page"
	"im/pkg/config"
	apigatewayService "im/server/apigateway/rpc/service"
	imGatewayService "im/server/imgateway/rpc/service"
	"image/color"
	"log/slog"
	"net"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// tokenAuth 实现了 credentials.PerRPCCredentials 接口
type tokenAuth struct {
	ctx    *common.Context
	logger *slog.Logger
}

// GetRequestMetadata 为每个 RPC 请求获取并设置认证元数据
func (t *tokenAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	t.logger.Debug("GetRequestMetadata", "uri", uri, "token", t.ctx.Token)
	return map[string]string{
		"token": t.ctx.Token,
	}, nil
}

// RequireTransportSecurity 指明是否需要安全的传输连接
func (t *tokenAuth) RequireTransportSecurity() bool {
	return false // 建议为 true 来配合 TLS
}

func main() {
	ctx := &common.Context{
		SessionUserTable:make(map[string]map[string]common.User, 0),
	}
	conf := config.NewConf().GetClientConfig()

	level := slog.LevelInfo
	if conf.Mode == "debug" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))

	apiGatewayConn, err := grpc.NewClient(conf.APIGatewayAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithPerRPCCredentials(&tokenAuth{ctx: ctx, logger: logger}))
	if err != nil {
		logger.Error("failed to create client", "error", err)
		return
	}
	apiGatewayClient := apigatewayService.NewAPIGatewayClient(apiGatewayConn)

	imGatewayConn, err := grpc.NewClient(conf.IMGatewayAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("failed to create client", "error", err)
		return
	}
	imGatewayClient := imGatewayService.NewIMGatewayClient(imGatewayConn)

	imGatewayLongConn, err := net.Dial("tcp", conf.IMGatewayAddr)
	if err != nil {
		logger.Error("failed to dial", "error", err)
		return
	}
	defer imGatewayLongConn.Close()

	a := app.New()

	// 应用自定义主题以隐藏滚动条
	a.Settings().SetTheme(&customTheme{Theme: theme.DefaultTheme()})

	ctx.App = a
	ctx.Ctx = context.Background()
	ctx.Logger = logger
	ctx.Config = conf
	ctx.ApiGatewayClient = apiGatewayClient
	ctx.IMGatewayClient = imGatewayClient
	ctx.IMGatewayLongConn = imGatewayLongConn
	messageReadChan := make(chan common.ChatMessage)
	messageWriteChan := make(chan common.ChatMessage)
	ctx.MessageReadChan = messageReadChan
	ctx.MessageWriteChan = messageWriteChan
	go common.Read(ctx)
	go common.Write(ctx)
	ctx.LoginPage = page.LoginPage(ctx)
	ctx.LoginPage.Show()
	a.Run()

}

// customTheme 自定义主题，用于隐藏滚动条
type customTheme struct {
	fyne.Theme
}

// Color 重写颜色方法，将滚动条设置为透明
func (ct *customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// 隐藏滚动条：将滚动条背景和前景色设为透明
	if name == theme.ColorNameScrollBar {
		return color.Transparent
	}
	// 其他颜色使用默认主题
	return theme.DefaultTheme().Color(name, variant)
}
