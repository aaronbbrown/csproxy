package main

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"
)

func checkError(err error) {
	if err != nil {
		log.Fatalf("Fatal error: %v", err.Error())
		os.Exit(1)
	}
}

func main() {
	viper.SetConfigName("csproxy")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	checkError(err)

	// Hacky workaround due to https://github.com/spf13/viper/issues/158
	if viper.Get("listeners.carbon.address") == nil {
		viper.Set("listeners.carbon.address", "127.0.0.1")
	}
	if viper.Get("listeners.carbon.port") == nil {
		viper.Set("listeners.carbon.port", 2003)
	}
	if viper.Get("listeners.http.address") == nil {
		viper.Set("listeners.http.address", "127.0.0.1")
	}
	if viper.Get("listeners.http.port") == nil {
		viper.Set("listeners.http.port", 9080)
	}

	done := make(chan bool, 1)

	var metricChannels []chan *Metric

	if viper.Get("writers.statsd.address") != nil &&
		viper.Get("writers.statsd.port") != nil {
		statsdMetrics := make(chan *Metric)
		metricChannels = append(metricChannels, statsdMetrics)

		var statsdTransforms []transform

		if viper.Get("transforms.statsd") != nil {
			viper.UnmarshalKey("transforms.statsd", &statsdTransforms)
			statsdTransforms = compileTransforms(statsdTransforms)
			fmt.Println(statsdTransforms)
		}

		go statsdWriter(
			viper.GetString("writers.statsd.address"),
			viper.GetInt("writers.statsd.port"),
			done, statsdMetrics, statsdTransforms)
	}

	if viper.Get("writers.carbon.address") != nil &&
		viper.Get("writers.carbon.port") != nil {
		carbonMetrics := make(chan *Metric)
		metricChannels = append(metricChannels, carbonMetrics)

		go carbonWriter(
			viper.GetString("writers.carbon.address"),
			viper.GetInt("writers.carbon.port"),
			done, carbonMetrics)
	}

	go carbonListener(
		viper.GetString("listeners.carbon.address"),
		viper.GetInt("listeners.carbon.port"),
		done, metricChannels)

	// for status
	go httpListener(
		viper.GetString("listeners.http.address"),
		viper.GetInt("listeners.http.port"),
		done)

	<-done
}
