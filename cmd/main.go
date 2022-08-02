package main

import (
	"flag"
	"github.com/AleksK1NG/go-elasticsearch/config"
	"github.com/AleksK1NG/go-elasticsearch/internal/app"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	"log"
)

func main() {
	log.Println("Starting microservice")

	flag.Parse()

	cfg, err := config.InitConfig()
	if err != nil {
		log.Fatal(err)
	}

	appLogger := logger.NewAppLogger(cfg.Logger)
	appLogger.InitLogger()
	appLogger.Named(app.GetMicroserviceName(cfg))
	appLogger.Infof("CFG: %+v", cfg)
	appLogger.Fatal(app.NewApp(appLogger, cfg).Run())
}
