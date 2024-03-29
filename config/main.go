package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/justwatchcom/elasticsearch_exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
)

func main() {
	var (
		Name                    = "elasticsearch_exporter"
		listenAddress           = flag.String("web.listen-address", ":9113", "Address to listen on for web interface and telemetry.")
		metricsPath             = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		esURI                   = flag.String("es.uri", "http://localhost:9200", "HTTP API address of an Elasticsearch node.")
		esTimeout               = flag.Duration("es.timeout", 5*time.Second, "Timeout for trying to get stats from Elasticsearch.")
		esAllNodes              = flag.Bool("es.all", false, "Export stats for all nodes in the cluster. If used, this flag will override the flag es.node.")
		esNode                  = flag.String("es.node", "_local", "Node's name of which metrics should be exposed.")
		esExportIndices         = flag.Bool("es.indices", false, "Export stats for indices in the cluster.")
		esExportClusterSettings = flag.Bool("es.cluster_settings", false, "Export stats for cluster settings.")
		esExportShards          = flag.Bool("es.shards", false, "Export stats for shards in the cluster (implies es.indices=true).")
		esExportSnapshots       = flag.Bool("es.snapshots", false, "Export stats for the cluster snapshots.")
		esCA                    = flag.String("es.ca", "", "Path to PEM file that contains trusted Certificate Authorities for the Elasticsearch connection.")
		esClientPrivateKey      = flag.String("es.client-private-key", "", "Path to PEM file that contains the private key for client auth when connecting to Elasticsearch.")
		esClientCert            = flag.String("es.client-cert", "", "Path to PEM file that contains the corresponding cert for the private key to connect to Elasticsearch.")
		esInsecureSkipVerify    = flag.Bool("es.ssl-skip-verify", false, "Skip SSL verification when connecting to Elasticsearch.")
		logLevel                = flag.String("log.level", "info", "Sets the loglevel. Valid levels are debug, info, warn, error")
		logFormat               = flag.String("log.format", "logfmt", "Sets the log format. Valid formats are json and logfmt")
		logOutput               = flag.String("log.output", "stdout", "Sets the log output. Valid outputs are stdout and stderr")
		showVersion             = flag.Bool("version", false, "Show version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Print(version.Print(Name))
		os.Exit(0)
	}

	logger := getLogger(*logLevel, *logOutput, *logFormat)

	esURIEnv, ok := os.LookupEnv("ES_URI")
	if ok {
		*esURI = esURIEnv
	}
	esURL, err := url.Parse(*esURI)
	if err != nil {
		_ = level.Error(logger).Log(
			"msg", "failed to parse es.uri",
			"err", err,
		)
		os.Exit(1)
	}

	// returns nil if not provided and falls back to simple TCP.
	tlsConfig := createTLSConfig(*esCA, *esClientCert, *esClientPrivateKey, *esInsecureSkipVerify)

	httpClient := &http.Client{
		Timeout: *esTimeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
		},
	}

	// version metric
	versionMetric := version.NewCollector(Name)
	prometheus.MustRegister(versionMetric)
	prometheus.MustRegister(collector.NewClusterHealth(logger, httpClient, esURL))
	prometheus.MustRegister(collector.NewNodes(logger, httpClient, esURL, *esAllNodes, *esNode))
	if *esExportIndices || *esExportShards {
		prometheus.MustRegister(collector.NewIndices(logger, httpClient, esURL, *esExportShards))
	}
	if *esExportSnapshots {
		prometheus.MustRegister(collector.NewSnapshots(logger, httpClient, esURL))
	}
	if *esExportClusterSettings {
		prometheus.MustRegister(collector.NewClusterSettings(logger, httpClient, esURL))
	}
	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err = w.Write([]byte(`<html>
			<head><title>Elasticsearch Exporter</title></head>
			<body>
			<h1>Elasticsearch Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
		if err != nil {
			_ = level.Error(logger).Log(
				"msg", "failed handling writer",
				"err", err,
			)
		}
	})

	_ = level.Info(logger).Log(
		"msg", "starting elasticsearch_exporter",
		"addr", *listenAddress,
	)

	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		_ = level.Error(logger).Log(
			"msg", "http server quit",
			"err", err,
		)
	}
}
