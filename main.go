package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/go-kit/kit/log/level"
	"github.com/hanazuki/prometheus-natureremo-exporter/natureremo"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	promlog_flag "github.com/prometheus/common/promlog/flag"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	var (
		listenAddress = kingpin.Flag("web.listen-address", "Address to listen on for telemetry.").Default(":9539").String()
		metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()

		natureremoEndpoint    = kingpin.Flag("natureremo.endpoint", "Nature Remo API endpoint.").Default(natureremo.DEFAULT_ENDPOINT).String()
		natureremoAccessToken = kingpin.Flag("natureremo.access-token", "Nature Remo API access token.").PlaceHolder("[file:PATH|env:VAR]").Required().String()
	)

	promlogConfig := &promlog.Config{}
	promlog_flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	endpoint, err := url.Parse(*natureremoEndpoint)
	if err != nil {
		level.Error(logger).Log("msg", fmt.Sprintf("Invalid value for --natureremo.endpoint: %s\n", err.Error()))
		os.Exit(1)
	}

	accessToken, err := readAccessToken(*natureremoAccessToken)
	if err != nil {
		level.Error(logger).Log("msg", fmt.Sprintf("Invalid value for --natureremo.access-token, %s\n", err.Error()))
		os.Exit(1)
	}

	httpClient := cleanhttp.DefaultClient()
	httpClient.Transport = HttpLogger{
		Transport: httpClient.Transport,
		Logger:    logger,
	}

	natureremo := natureremo.Client{
		HttpClient:  httpClient,
		Endpoint:    *endpoint,
		AccessToken: accessToken,
	}

	exporter := NewExporter(natureremo, logger)
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, promhttp.Handler())
	http.ListenAndServe(*listenAddress, nil)
}

func readAccessToken(spec string) (string, error) {
	if strings.HasPrefix(spec, "env:") {
		name := spec[4:]
		value, ok := os.LookupEnv(name)
		if !ok {
			return "", fmt.Errorf("Environment variable %s is not set.", name)
		}
		return value, nil
	} else if strings.HasPrefix(spec, "file:") {
		path := spec[5:]
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return "", err
		}
		return strings.TrimSuffix(string(content), "\n"), nil
	}
	return "", fmt.Errorf("Unknown specification `%s'", spec)
}
