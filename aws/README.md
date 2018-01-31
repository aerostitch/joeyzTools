# AWS-related scripts

## `aws_iam_get_user_from_access_key.go`

This script either list all keys on an account
or list the user corresponding to the given key

Parameters:
* `-filter-key` (default: ""): Aws access key to look for. If not provided, will list all of them.

Usage example:
```
go run aws_iam_get_user_from_access_key.go -filter-key AKIABCDEFGHIJKLMNOPQ
```

## `aws_ec2_security_groups_opened.go`

This script list the security groups with ports opened to the world for IPv4.

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

## `elb_log_analyzer`

The script in `elb_log_analyzer` is a tool to parse AWS Elastic LoadBalancer
access logs and push them to a MySQL database.
