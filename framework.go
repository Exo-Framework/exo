package exo

import (
	"context"
	"os"
	"os/signal"
	"strings"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	_ "github.com/joho/godotenv/autoload"
)

// Framework is a struct that holds the fiber.App instance and the configuration for the exo framework.
type Framework struct {
	*fiber.App
	config Config
}

// New creates a new instance of the exo framework.
// The instance is a fiber.App instance with custom JSON encoder and decoder and some default configurations.
func New(opts ...ConfigOption) *Framework {
	config := getConfig(opts)

	app := &Framework{fiber.New(fiber.Config{
		ErrorHandler: config.errorHandler,
		JSONDecoder: func(data []byte, v interface{}) error {
			return json.Unmarshal(data, v)
		},
		JSONEncoder: func(v interface{}) ([]byte, error) {
			return json.Marshal(v)
		},
	}), config}

	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	app.Use(helmet.New())

	if config.logging {
		app.Use(logger.New())
	}

	if config.compress {
		app.Use(compress.New())
	}

	if config.cors.enable {
		app.Use(cors.New(cors.Config{
			AllowOrigins:     strings.Join(config.cors.allowOrigins, ", "),
			AllowMethods:     strings.Join(config.cors.allowMethods, ", "),
			AllowHeaders:     strings.Join(config.cors.allowHeaders, ", "),
			AllowCredentials: config.cors.allowCredentials,
			ExposeHeaders:    strings.Join(config.cors.exposeHeaders, ", "),
			MaxAge:           config.cors.maxAge,
		}))
	}

	if config.etag != nil {
		app.Use(etag.New(etag.Config{
			Weak: *config.etag,
		}))
	}

	if config.cache != nil {
		app.Use(cache.New(cache.Config{
			Expiration:   config.cache.Expiration,
			CacheControl: config.cache.ClientControl,
		}))
	}

	return app
}

// Start starts the server. It will block the current goroutine and fatal out if the server fails to start.
// It will also listen for the interrupt signal and gracefully shutdown the server upon receiving SIGINT.
// The DB's Connect function must be called before calling Start.
func (f *Framework) Start() {
	f.initializeDB()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go f.startHTTP()

	<-ctx.Done()

	if err := f.Shutdown(); err != nil {
		log.Fatal(err)
	}
}

func (f *Framework) startHTTP() {
	if f.config.ssl != nil {
		log.Fatal(f.ListenTLSWithCertificate(f.config.addr(), *f.config.ssl))
	} else {
		log.Fatal(f.Listen(f.config.addr()))
	}
}

func (f *Framework) initializeDB() {
	if f.config.db != nil {
		return
	}
}
