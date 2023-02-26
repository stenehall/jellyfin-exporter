package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jessevdk/go-flags"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var (
	Version = "unknown"
	log     *logrus.Entry
)

type ExporterConfig struct {
	LogLevel  string `long:"log-level" description:"log verbosity level (trace, debug, info, warn, error, fatal)" env:"LOG_LEVEL" default:"info"`
	Namespace string `long:"namespace" description:"metric name prefix" default:"jellyfin" env:"METRIC_NAMESPACE"`
	Listen    string `short:"l" long:"listen" description:"host:port to listen on" default:":9453" env:"LISTEN"`
	Host      string `short:"h" long:"host" description:"jellyfin host to export metrics for" required:"true" env:"HOST"`
	APIKey    string `short:"u" long:"apikey" description:"jellyfin apikey for auth" required:"true" env:"API_KEY"`
}

type JellyfinGetCollector struct {
	Config *ExporterConfig

	version     *prom.Desc
	movieCount  *prom.Desc
	seriesCount *prom.Desc
}

func init() {
	log = logrus.WithContext(context.Background())
	log.Logger.SetOutput(os.Stderr)
	log.Logger.Formatter = &prefixed.TextFormatter{
		FullTimestamp:  true,
		QuoteCharacter: "'",
	}
}

func main() {
	var config ExporterConfig
	parser := flags.NewParser(&config, flags.HelpFlag|flags.PassDoubleDash)
	parser.Groups()[0].ShortDescription = "Options"
	_, err := parser.Parse()
	if err != nil {
		var flagsErr *flags.Error
		if errors.As(err, &flagsErr) && flagsErr.Type == flags.ErrHelp {
			fmt.Fprintf(os.Stderr, "Jellyfin Exporter (version %s)\n\n", Version)
			parser.WriteHelp(os.Stderr)
			os.Exit(0)
		} else {
			log.WithError(err).
				Fatal("parse flags")
		}
	}

	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		log.WithError(err).Warnf("invalid log level")
	} else {
		log.Logger.SetLevel(level)
	}
	log.Info("jellyfin-exporter version " + Version)

	collector := NewJellyfinGetCollector(&config)
	prom.MustRegister(collector)

	// Test if the host responds
	var response struct {
		Version string `json:"version"`
	}
	err = collector.getAPI("/System/Info", &response)
	if err != nil {
		log.WithError(err).Warn("failed to get jellyfin version")
	} else {
		log.Infof("jellyfin version %s", response.Version)
	}

	promHandler := promhttp.Handler()
	var metrics http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		log.WithField("remote", r.RemoteAddr).
			Info(fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		promHandler.ServeHTTP(w, r)
	}

	var health http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		log.Info("Healthcheck status ok")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.WithError(err).Panic("healthcheck failed")
		}
	}

	log.Info("serving metrics at " + config.Listen)

	http.Handle("/metrics", metrics)
	http.Handle("/_health", health)

	err = http.ListenAndServe(config.Listen, nil) //nolint:gosec
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.WithError(err).Panic("listenandserve")
	}
}

func NewJellyfinGetCollector(config *ExporterConfig) *JellyfinGetCollector {
	return &JellyfinGetCollector{
		Config: config,

		version: prom.NewDesc(
			prom.BuildFQName(config.Namespace, "", "version"),
			"always 1. label 'version' contains Jellyfin server version",
			[]string{"version"}, nil,
		),
		movieCount: prom.NewDesc(
			prom.BuildFQName(config.Namespace, "", "movieCount"),
			"Number of movies in the Library",
			nil, nil,
		),
		seriesCount: prom.NewDesc(
			prom.BuildFQName(config.Namespace, "", "seriesCount"),
			"Number of series in the Library",
			nil, nil,
		),
	}
}

func (c *JellyfinGetCollector) getAPI(endpoint string, out interface{}) error {
	host := strings.TrimRight(c.Config.Host, "/")

	u, err := url.Parse(host + endpoint)
	if err != nil {
		return err
	}
	log.WithField("url", u.String()).Debug("GET api")

	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Emby-Token", c.Config.APIKey)
	// @todo: fix this
	resp, err := netClient.Do(req) //nolint:bodyclose
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jellyfin api response %d %s",
			resp.StatusCode, http.StatusText(resp.StatusCode),
		)
	}
	err = json.NewDecoder(resp.Body).Decode(out)
	if err != nil {
		return err
	}

	return nil
}

func (c *JellyfinGetCollector) Collect(metrics chan<- prom.Metric) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		var count map[string]float64
		err := c.getAPI("/Items/Counts", &count)
		if err != nil {
			panic(err)
		}

		metrics <- prom.MustNewConstMetric(c.movieCount, prom.GaugeValue, count["MovieCount"])
		metrics <- prom.MustNewConstMetric(c.seriesCount, prom.GaugeValue, count["SeriesCount"])
	}()

	go func() {
		defer wg.Done()
		var response struct {
			Version string `json:"version"`
		}
		err := c.getAPI("/System/Info", &response)
		if err != nil {
			panic(err)
		}
		metrics <- prom.MustNewConstMetric(c.version, prom.GaugeValue, 1, response.Version)
	}()

	wg.Wait()
}

func (c *JellyfinGetCollector) Describe(descr chan<- *prom.Desc) {
	descr <- c.version
	descr <- c.movieCount
	descr <- c.seriesCount
}
