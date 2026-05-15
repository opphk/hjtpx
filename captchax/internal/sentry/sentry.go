package sentry

import (
	"log"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

type Config struct {
	DSN              string  `yaml:"dsn"`
	Environment      string  `yaml:"environment"`
	Release          string  `yaml:"release"`
	TracesSampleRate float64 `yaml:"traces_sample_rate"`
	Debug            bool    `yaml:"debug"`
}

func Init(cfg Config) error {
	if cfg.DSN == "" {
		log.Println("⚠️ Sentry DSN not configured, Sentry will not start")
		return nil
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		TracesSampleRate: cfg.TracesSampleRate,
		Debug:            cfg.Debug,
		AttachStacktrace: true,
	})
	if err != nil {
		return err
	}

	log.Println("✅ Sentry initialized")
	return nil
}

func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

func CaptureException(err error, tags map[string]string) {
	hub := sentry.CurrentHub().Clone()
	for key, value := range tags {
		hub.Scope().SetTag(key, value)
	}
	hub.CaptureException(err)
}

func CaptureMessage(message string, tags map[string]string) {
	hub := sentry.CurrentHub().Clone()
	for key, value := range tags {
		hub.Scope().SetTag(key, value)
	}
	hub.CaptureMessage(message)
}

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		hub := sentry.GetHubFromContext(c.Request.Context())
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
		}

		hub.Scope().SetRequest(c.Request)
		ctx := sentry.SetHubOnContext(c.Request.Context(), hub)

		span := sentry.StartSpan(ctx, "http.server",
			sentry.WithTransactionName(c.Request.Method+" "+c.FullPath()),
		)
		defer span.Finish()

		c.Request = c.Request.WithContext(span.Context())

		c.Next()

		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				hub.CaptureException(err.Err)
			}
		}
	}
}
