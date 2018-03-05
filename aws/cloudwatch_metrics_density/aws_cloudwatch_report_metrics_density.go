package main

/*
This script reports the number of metrics in cloudwatch for each namespace /
metric / metric dimension that can be pulled over the last 2 hours. It helps
tracking the density of the metrics and this way find who stores at least 1
metric per second. This is needed when you try to track down the spend in the
monthly report for the PutMetrics operation.

Outputs the result in a CSV format in 3 different provided files:
* `-detailed-file` is the file where the number of datapoints for each Namespace/Metric/Dimension name/Dimension value will be stored
* `-metrics-file` is the file where the number of datapoints for each Namespace/Metric will be stored
* `-metrics-file` is the file where the number of datapoints for each Namespace will be stored

Usage example:
  go run aws_report_cloudwatch_metrics_density.go  -nb-workers 32 -detailed-file /tmp/${AWS_PROFILE}_cw_density.csv -metrics-file /tmp/${AWS_PROFILE}_cw_metrics.csv -namespaces-file /tmp/${AWS_PROFILE}_cw_ns.csv

Note: this script does not differentiate native metrics from custom or enhanced metrics.
*/

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/gobike/envflag"
)

type metricBase struct {
	Namespace, MetricName, DimensionName, DimensionValue string
}
type metricInput struct {
	Namespace, MetricName, DimensionName, DimensionValue *string
	StartTime, EndTime                                   *time.Time
}
type metricOutput struct {
	Namespace, MetricName, DimensionName, DimensionValue *string
	DPCount                                              uint
}

// CWProcessor is used to share the Cloudwatch client between functions
type CWProcessor struct {
	svc          cloudwatchiface.CloudWatchAPI
	wg, wgagg    sync.WaitGroup
	metricsList  chan *metricInput
	metricsCount chan *metricOutput
}

// newCWProcessor initiates a new cloudwatch client
func newCWProcessor() *CWProcessor {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	return &CWProcessor{svc: cloudwatch.New(sess), metricsList: make(chan *metricInput), metricsCount: make(chan *metricOutput)}
}

// initWorkerPool starts the different workers that will count and aggregate the
// datapoints
func (c *CWProcessor) initWorkerPool(nbWorkers uint, rawDataFile, metricsAggregateFile, namespaceAggregateFile string) {
	for i := uint(0); i < nbWorkers; i++ {
		c.wg.Add(1)
		go c.dataPointCounter()
	}
	c.wgagg.Add(1)
	go c.dataPointAggregator(rawDataFile, metricsAggregateFile, namespaceAggregateFile)
}

// wait waits that the worker finish their work
func (c *CWProcessor) wait() {
	c.wg.Wait()
	close(c.metricsCount)
	c.wgagg.Wait()
}

// waitForQueue waits that the metricsList channel is less than the given maxLen
func (c *CWProcessor) waitForQueue(maxLen int) {
	for {
		if len(c.metricsList) > maxLen {
			time.Sleep(100 * time.Millisecond)
		} else {
			return
		}
	}
}

// processMetrics pulls the list of metrics and the number of datapoints it can
// get over the last 2 hours (minimum granularity = 1h). The goal is to have a
// view of how dense the data is sent by namespace, metric & metric dimension
func (c *CWProcessor) processMetrics() {
	params := &cloudwatch.ListMetricsInput{}
	timeShift := 1440 // 2h = 7200 seconds. You can only pull 1440 data points at a time
	timeMax := 7200
	err := c.svc.ListMetricsPages(params,
		func(page *cloudwatch.ListMetricsOutput, lastPage bool) bool {
			for _, m := range page.Metrics {
				for _, d := range m.Dimensions {
					// Wait if the queue is too large to avoid having the workers
					// query for high density metrics outside of the high density window
					c.waitForQueue(30)
					timeIndex := 0
					timeRef := time.Now().UTC()
					for timeIndex < timeMax {
						end := timeRef.Add(-time.Duration(timeIndex) * time.Second)
						if (timeIndex + timeShift) > timeMax {
							timeShift = timeMax - timeIndex
						}
						start := end.Add(-time.Duration(timeShift) * time.Second)
						input := metricInput{MetricName: m.MetricName, Namespace: m.Namespace, DimensionName: d.Name, DimensionValue: d.Value, StartTime: &start, EndTime: &end}
						c.metricsList <- &input
						timeIndex += timeShift
					}
				}
			}
			return !lastPage
		})
	close(c.metricsList)
	if err != nil {
		fmt.Println("Error", err)
		return
	}
}

// dataPointCounter is an worker that retrieves the count of datapoints from the
// metrics fed through the channel
func (c *CWProcessor) dataPointCounter() {
	period := int64(1)
	for m := range c.metricsList {
		output := c.getDatapointsCount(m.Namespace, m.MetricName, m.DimensionName, m.DimensionValue, m.StartTime, m.EndTime, &period)
		data := metricOutput{Namespace: m.Namespace, MetricName: m.MetricName, DimensionName: m.DimensionName, DimensionValue: m.DimensionValue, DPCount: output}
		c.metricsCount <- &data
	}
	c.wg.Done()
}

// getDatapointsCount returns the number of datapoints over the given time
// interval using the period. Be careful, there should not be more than 1440
// datapoints if you don't want the aws call to fail.
func (c *CWProcessor) getDatapointsCount(namespace, metricName, dimensionName, dimensionValue *string, start, end *time.Time, period *int64) uint {
	params := cloudwatch.GetMetricStatisticsInput{
		Dimensions: []*cloudwatch.Dimension{{Name: dimensionName, Value: dimensionValue}},
		MetricName: metricName,
		Namespace:  aws.String(*namespace),
		StartTime:  start,
		Period:     period,
		EndTime:    end,
		Statistics: []*string{aws.String("Sum")},
	}
	result, err := c.svc.GetMetricStatistics(&params)
	if err != nil {
		fmt.Println(err)
	}
	return uint(len(result.Datapoints))
}

// dataPointAggregator is a worker that aggregates the metrics counts to
// generate the reporting
func (c *CWProcessor) dataPointAggregator(rawDataFile, metricsAggregateFile, namespaceAggregateFile string) {
	dpAgg := make(map[metricBase]uint)
	dpMetrics := make(map[string]uint)
	dpNs := make(map[string]uint)
	for m := range c.metricsCount {
		k := metricBase{Namespace: *m.Namespace, MetricName: *m.MetricName, DimensionName: *m.DimensionName, DimensionValue: *m.DimensionValue}
		ns := *m.Namespace
		met := fmt.Sprintf("%s,%s", *m.Namespace, *m.MetricName)
		val := m.DPCount
		if _, ok := dpNs[ns]; ok {
			dpNs[ns] += val
		} else {
			dpNs[ns] = val
		}
		if _, ok := dpMetrics[met]; ok {
			dpMetrics[met] += val
		} else {
			dpMetrics[met] = val
		}
		if _, ok := dpAgg[k]; ok {
			dpAgg[k] += val
		} else {
			dpAgg[k] = val
		}
	}
	if f, err := os.Create(namespaceAggregateFile); err != nil {
		fmt.Println(err)
	} else {
		f.WriteString("Namespace,Number of datapoints\n")
		for k, v := range dpNs {
			f.WriteString(fmt.Sprintf("%s,%d\n", k, v))
		}
		f.Sync()
		f.Close()
	}
	if f, err := os.Create(metricsAggregateFile); err != nil {
		fmt.Println(err)
	} else {
		f.WriteString("Namespace,Metric,Number of datapoints\n")
		for k, v := range dpMetrics {
			f.WriteString(fmt.Sprintf("%s,%d\n", k, v))
		}
		f.Sync()
		f.Close()
	}
	if f, err := os.Create(rawDataFile); err != nil {
		fmt.Println(err)
	} else {
		f.WriteString("Namespace,Metric,Dimension Name,Dimension Value,Number of datapoints\n")
		for k, v := range dpAgg {
			f.WriteString(fmt.Sprintf("%s,%s,%s,%s,%d\n", k.Namespace, k.MetricName, k.DimensionName, k.DimensionValue, v))
		}
		f.Sync()
		f.Close()
	}
	c.wgagg.Done()
}

func main() {
	var (
		nbWorkers                                                 uint
		rawDataFile, metricsAggregateFile, namespaceAggregateFile string
	)
	flag.UintVar(&nbWorkers, "nb-workers", 5, "Number of workers used to fetch the metrics datapoints. Env variable: NB_WORKERS")
	flag.StringVar(&rawDataFile, "detailed-file", "cloudwatch_datapoints_density.csv", "Path (including the file name) of the CSV file containing the detailed statistics on the number of datapoints per namespace/metric/dimension. Environment variable: DETAILED_FILE")
	flag.StringVar(&metricsAggregateFile, "metrics-file", "cloudwatch_metrics_datapoints_density.csv", "Path (including the file name) of the CSV file containing the aggregated statistics on the number of datapoints per namespace/metric. Environment variable: METRICS_FILE")
	flag.StringVar(&namespaceAggregateFile, "namespaces-file", "cloudwatch_namespace_datapoints_density.csv", "Path (including the file name) of the CSV file containing the aggregated statistics on the number of datapoints per namespace. Environment variable: NAMESPACES_FILE")
	envflag.Parse()

	p := newCWProcessor()
	p.initWorkerPool(nbWorkers, rawDataFile, metricsAggregateFile, namespaceAggregateFile)
	p.processMetrics()
	p.wait()
}
