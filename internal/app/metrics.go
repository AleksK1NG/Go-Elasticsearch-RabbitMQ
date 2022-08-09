package app

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (a *app) runMetrics(cancel context.CancelFunc) {
	a.metricsServer = echo.New()
	go func() {
		a.metricsServer.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
			StackSize:         stackSize,
			DisablePrintStack: false,
			DisableStackAll:   false,
		}))

		a.metricsServer.GET(a.cfg.Probes.PrometheusPath, echo.WrapHandler(promhttp.Handler()))
		//a.metricsServer.GET(a.cfg.Probes.LivenessPath, func(c echo.Context) error {
		//	a.log.Debugf("live healthcheck %s", c.Request().URL.Path)
		//	return c.JSON(http.StatusOK, "OK")
		//})
		//a.metricsServer.GET(a.cfg.Probes.ReadinessPath, func(c echo.Context) error {
		//	a.log.Debugf("ready healthcheck %s", c.Request().URL.Path)
		//	return c.JSON(http.StatusOK, "OK")
		//})

		a.log.Infof("Metrics app is running on port: %s", a.cfg.Probes.PrometheusPort)
		if err := a.metricsServer.Start(a.cfg.Probes.PrometheusPort); err != nil {
			a.log.Errorf("metricsServer.Start: %v", err)
			cancel()
		}
	}()
}
