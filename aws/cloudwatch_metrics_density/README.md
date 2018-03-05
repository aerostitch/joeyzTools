# Cloudwatch metrics density report

## `aws_cloudwatch_report_metrics_density.go`

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
```
export AWS_PROFILE=my_profile
go run aws_report_cloudwatch_metrics_density.go  -nb-workers 32 -detailed-file /tmp/${AWS_PROFILE}_cw_density.csv -metrics-file /tmp/${AWS_PROFILE}_cw_metrics.csv -namespaces-file /tmp/${AWS_PROFILE}_cw_ns.csv
```

Note: this script does not differentiate native metrics from custom or enhanced
metrics.

Arguments:
```
  -detailed-file string
        Path (including the file name) of the CSV file containing the detailed statistics on the number of datapoints per namespace/metric/dimension. Environment variable: DETAILED_FILE (default "cloudwatch_datapoints_density.csv")
  -metrics-file string
        Path (including the file name) of the CSV file containing the aggregated statistics on the number of datapoints per namespace/metric. Environment variable: METRICS_FILE (default "cloudwatch_metrics_datapoints_density.csv")
  -namespaces-file string
        Path (including the file name) of the CSV file containing the aggregated statistics on the number of datapoints per namespace. Environment variable: NAMESPACES_FILE (default "cloudwatch_namespace_datapoints_density.csv")
  -nb-workers uint
        Number of workers used to fetch the metrics datapoints. Env variable: NB_WORKERS (default 5)

```
