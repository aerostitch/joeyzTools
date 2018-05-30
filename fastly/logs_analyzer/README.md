# `fastly_logs_analyzer` script

This tool analyzes access logs from hls traffic hosted on fastly and pushes the result
inside a MySQL/MariaDB database so that you can generate some reporting on it.

Note that the report is centered around the bitrate and bytes usage of the traffic.

Easy way to get a DB:
```
docker run --name some-mariadb -e MYSQL_ROOT_PASSWORD=my-secret-pw -e MYSQL_DATABASE=accesslogs -p 3306:3306 -d mariadb:latest
```

To bulk load your local files in this case (without generating the standard reports):
```
DB_NAME=accesslogs
TBL=bla
go run fastly_logs_analyzer.go  -db-host "tcp(172.17.0.2)" \


	-db-name ${DB_NAME} \
	-db-user root \
	-db-pwd my-secret-pw \
	-db-table ${TBL} \
	-recursive \
	-file-path /tmp/${TBL}

```

Bulk loading from S3 and generate the standard report:
```
DB_NAME=accesslogs
TBL=bla
BUCKET_NAME=my-elb-logs-bucket
go run fastly_logs_analyzer.go  -db-host "tcp(172.17.0.2)" \


	-db-name ${DB_NAME} \
	-db-user root \
	-db-pwd my-secret-pw \
	-db-table ${TBL} \
	-s3-bucket ${BUCKET_NAME} \
	-s3-path ${TBL}/AWSLogs

```

Bulk loading from local files and generate the standard report:
```
DB_NAME=accesslogs
TBL=bla
go run fastly_logs_analyzer.go  -db-host "tcp(172.17.0.2)" \


	-db-name ${DB_NAME} \
	-db-user root \
	-db-pwd my-secret-pw \
	-db-table ${TBL} \
	-recursive \
	-file-path /tmp/${TBL} \
	-report-path /tmp/${TBL}_summary.csv

```

Only generate the standard report:
```
DB_NAME=accesslogs
TBL=bla
go run fastly_logs_analyzer.go  -db-host "tcp(172.17.0.2)" \


	-db-name ${DB_NAME} \
	-db-user root \
	-db-pwd my-secret-pw \
	-db-table ${TBL} \
	-report-path /tmp/${TBL}_summary.csv

```

Note that you can also go into your DB and generate your own custom reports...
