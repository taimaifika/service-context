package cmd

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"gorm.io/gorm"

	"github.com/taimaifika/service-context/component/ginc"
	"github.com/taimaifika/service-context/component/gormc"
	"github.com/taimaifika/service-context/component/otelc"
	"github.com/taimaifika/service-context/component/slogc"

	sctx "github.com/taimaifika/service-context"
)

var serviceContextName = "service-context-gorm"

func newServiceCtx() sctx.ServiceContext {
	return sctx.NewServiceContext(
		sctx.WithName(serviceContextName),
		sctx.WithComponent(slogc.NewSlogComponent()),
		sctx.WithComponent(otelc.NewOtel("otel")),
		sctx.WithComponent(ginc.NewGin("gin")),
		sctx.WithComponent(gormc.NewGormDB("postgres", "postgres")),
	)
}

type OtelComponent interface {
	GetTracerProvider() *trace.TracerProvider
}

type GINginComponent interface {
	GetPort() int
	GetRouter() *gin.Engine
}

type GormComponent interface {
	GetDB() *gorm.DB
}

type pgRepo struct {
	db *gorm.DB
}

func NewPgRepo(db *gorm.DB) *pgRepo {
	return &pgRepo{db: db}
}

var rootCmd = &cobra.Command{
	Use:   "app",
	Short: "Start GIN-HTTP service",
	Run: func(cmd *cobra.Command, args []string) {
		serviceCtx := newServiceCtx()

		if err := serviceCtx.Load(); err != nil {
			log.Fatal(err)
		}

		ginComp := serviceCtx.MustGet("gin").(GINginComponent)

		router := ginComp.GetRouter()

		router.Use(
			gin.Recovery(),
			gin.Logger(), // format log to text
			otelgin.Middleware(serviceContextName),
		)

		router.GET("/ping", func(c *gin.Context) {
			slog.InfoContext(c, "This is an info message", slog.String("key", "value"))

			slog.Debug("This is a debug message")

			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})

		// gorm component
		db := serviceCtx.MustGet("postgres").(GormComponent)
		pgRepo := NewPgRepo(db.GetDB())

		router.GET("/number", func(c *gin.Context) {
			ctx, span := otel.Tracer("service-context-gorm").Start(c.Request.Context(), "get-tasks")
			defer span.End()

			var num int
			if err := pgRepo.db.WithContext(ctx).Raw("SELECT 42").Scan(&num).Error; err != nil {
				panic(err)
			}

			slog.Info("Number", slog.Int("number", num))

			c.JSON(http.StatusOK, gin.H{"data": num})
		})

		// start the server
		if err := router.Run(fmt.Sprintf(":%d", ginComp.GetPort())); err != nil {
			log.Fatal(err)
		}
	},
}

func Execute() {
	rootCmd.AddCommand(outEnvCmd)
	slog.Info("Starting application")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
