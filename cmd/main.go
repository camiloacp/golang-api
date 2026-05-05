package main

import (
	"context"
	"errors"
	"golang-api/authorization"
	"golang-api/handler"
	"golang-api/storage"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

func main() {
	err := authorization.LoadFiles("certificates/private.pem", "certificates/public.pem")
	if err != nil {
		log.Fatalf("error loading certificates: %v", err)
	}

	// Inicialización de dependencias: conexión a BD + migración de schema.
	// Si cualquiera falla aquí, no tiene sentido arrancar el server.
	store := storage.New(storage.PostgreSQL)
	if err := store.Migrate(); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	e := echo.New()

	// Middlewares globales — corren en TODAS las rutas, en este orden:
	//   1. Recover:       atrapa panics y devuelve 500 limpio.
	//   2. RequestLogger: loguea cada request (método, URI, status, latencia).
	//   3. BodyLimit:     rechaza bodies > 1 MiB (reemplaza http.MaxBytesReader).
	e.Use(echomw.Recover())
	e.Use(echomw.RequestLoggerWithConfig(echomw.RequestLoggerConfig{
		LogStatus:  true,
		LogURI:     true,
		LogMethod:  true,
		LogLatency: true,
		LogValuesFunc: func(c echo.Context, v echomw.RequestLoggerValues) error {
			log.Printf("method=%s uri=%s status=%d latency=%s",
				v.Method, v.URI, v.Status, v.Latency)
			return nil
		},
	}))
	e.Use(echomw.BodyLimit("1M"))

	handler.RouteLogin(e, store)
	handler.RoutePerson(e, store)
	handler.RouteSignup(e, store)

	// Usamos http.Server explícito (en vez del atajo http.ListenAndServe)
	// porque necesitamos acceder a Shutdown() para el graceful shutdown.
	// ReadHeaderTimeout protege contra ataques tipo Slowloris: clientes que
	// abren conexiones y mandan headers muy lento para agotar el pool.
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           e,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// NotifyContext devuelve un contexto que se cancela cuando llega cualquiera
	// de las señales indicadas. Reemplaza el patrón viejo de canales + signal.Notify.
	//   - SIGINT:  lo manda Ctrl+C en la terminal.
	//   - SIGTERM: lo manda Docker/Kubernetes al detener el contenedor.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ListenAndServe es bloqueante, así que lo corremos en una goroutine.
	// Así el main puede esperar en <-ctx.Done() para reaccionar a la señal.
	go func() {
		log.Println("Server is running on http://localhost:8080")
		// Cuando llamamos a srv.Shutdown(), ListenAndServe retorna ErrServerClosed.
		// Ese NO es un error real: es la señal de "cerré limpio". Por eso lo ignoramos.
		// Cualquier otro error sí es fatal (puerto ocupado, listener roto, etc.).
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// Bloqueamos el main hasta que llegue una señal de terminación.
	<-ctx.Done()
	log.Println("shutdown signal received, draining requests...")

	// Le damos 10s a las requests en vuelo para terminar antes de cortar.
	// Kubernetes manda SIGKILL 30s después del SIGTERM por default, así que
	// 10s deja margen cómodo. Si pasás ese timeout, Shutdown retorna error
	// y las requests restantes se cortan.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown: deja de aceptar conexiones nuevas y espera a las activas.
	// Si en el futuro agregás un store.Close(), va DESPUÉS de Shutdown
	// (las requests en vuelo pueden estar usando la BD).
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("server stopped cleanly")
}
