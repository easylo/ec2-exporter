package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/easylo/ec2-exporter/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
)

var (
	showVersion   = flag.Bool("version", false, "Print version information.")
	listenAddress = flag.String("web.listen-address", ":9599", "Address to listen on for web interface and telemetry.")
	metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	awsKey        = flag.String("aws.key", "", "AWS EC2 account Key")
	awsSecret     = flag.String("aws.secret", "", "AWS EC2 account Secret")
	awsRegions    = flag.String("aws.regions", "", "AWS EC2 Region")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Fprintln(os.Stdout, version.Print("ec2-exporter"))
		os.Exit(0)
	}

	key := *awsKey

	if *awsKey == "" {
		key = os.Getenv("AWS_ACCESS_KEY_ID")
		if key == "" {
			log.Fatal("The AWS account Key is missing")
			os.Exit(0)
		}
	}

	secret := *awsSecret

	if *awsSecret == "" {
		secret = os.Getenv("AWS_SECRET_ACCESS_KEY")
		if secret == "" {
			log.Fatal("The AWS account Key is missing")
			os.Exit(0)
		}
	}

	regions := strings.Split(*awsRegions, ",")

	e := exporter.New(key, secret, regions)

	prometheus.MustRegister(e)

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>AWS EC2 Exporter</title></head>
             <body>
             <h1>AWS EC2 Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	log.Infoln("Listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
