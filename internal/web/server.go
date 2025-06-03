package lwt

import (
	"context"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	echoSwagger "github.com/swaggo/echo-swagger"
	"sysafari.com/customs/cguard/internal/config"
)

var server *echo.Echo

// SetupRoutes 配置服务器路由
// @title LWT web service
// @version 1.0
// @description This is a simple web service that provides download lwt file
// @termsOfService http://swagger.io/terms/

// @contact.name Joker
// @contact.email ljr@y-clouds.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:7004
// @BasePath /v2
func SetupRoutes() *echo.Echo {
	e := echo.New()

	// swagger
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// api exp:
	// http://localhost:{port}/lwt/OP210603005_20220909153131.xlsx?download=1
	e.GET("/lwt/:filename", DownloadLwtExcel)

	server = e
	return e
}

// StartServer 启动Web服务器
func StartServer() error {
	if server == nil {
		server = SetupRoutes()
	}

	// 启动web服务器
	cfg := config.GetConfig()
	port := fmt.Sprintf("%d", cfg.Port)
	if port == "0" {
		port = "1324"
	}

	log.Infof("Web服务器启动在端口: %s", port)
	return server.Start(":" + port)
}

// ShutdownServer 优雅关闭Web服务器
func ShutdownServer(timeout time.Duration) error {
	if server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return server.Shutdown(ctx)
}
