/*
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

*/
package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gobike/envflag"
)

var wg, s3wg sync.WaitGroup
var filePattern = regexp.MustCompile(`^<\d+>([^ ]*)\s[^\s]+\ss3\/\/[\w|-]+\[\d+]:\s[-.0-9]*\s.*\[.*\]\s"\w+\s\/([\w|\d]+)\/.+[_/](\d+x\d+|index|subtitles)+?.*\sHTTP/[\d.]+"\s(\d{3})?\s(\d+|"-*")\s".*"\s"(.*?)"`)

type accessLogEntry struct {
	year, month, day, hour, bytes                int
	hlsVersion, bitrate, responseCode, userAgent string
}

// processLine takes a line and the compiled regex and returns a accessLogEntry
func processLine(re *regexp.Regexp, line string) *accessLogEntry {
	entry := accessLogEntry{}

	result := re.FindStringSubmatch(line)

	// skip lines with incorrect length
	if len(result) < 5 {
		fmt.Printf("Skipping line: %s\nwhich had the following result object: %#v\n", line, result)
		return nil
	}

	layout := "2006-01-02T15:04:05Z"
	mDate, err := time.Parse(layout, result[1])
	if err != nil {
		log.Println(err)
	}
	entry.year = mDate.Year()
	entry.month = int(mDate.Month())
	entry.day = mDate.Day()
	entry.hour = mDate.Hour()

	entry.hlsVersion = result[2]
	entry.bitrate = result[3]
	entry.responseCode = result[4]
	if i, err := strconv.Atoi(result[5]); err != nil {
		entry.bytes = 0
	} else {
		entry.bytes = i
	}
	entry.userAgent = result[6]

	return &entry
}

// processS3Files processes each file found in the given key
func processS3Files(bucket, path string, dataPipe chan *accessLogEntry) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	fchan := make(chan string)
	// s3 files are processed in parallel by groups of C5maxParallelFiles
	for i := 0; i <= 5; i++ {
		s3wg.Add(1)
		go processS3File(bucket, sess, dataPipe, fchan)
	}

	svc := s3.New(sess)
	params := &s3.ListObjectsInput{Bucket: &bucket, Prefix: &path}
	errLst := svc.ListObjectsPages(params, func(page *s3.ListObjectsOutput, lastPage bool) bool {
		for _, obj := range page.Contents {
			fchan <- *obj.Key
		}

		return !lastPage
	})
	if errLst != nil {
		log.Println(errLst)
	}
	close(fchan)
	s3wg.Wait()
}

// processS3File process a single s3 file and sends its content to the channel
func processS3File(bucket string, sess *session.Session, dataPipe chan *accessLogEntry, fchan chan string) {
	s3dl := s3manager.NewDownloader(sess)
	for path := range fchan {
		log.Printf("Processing s3 file: s3://%s/%s", bucket, path)
		buff := &aws.WriteAtBuffer{}
		_, err := s3dl.Download(buff, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(path),
		})
		if err != nil {
			log.Println(err)
		}

		rdr, err := gzip.NewReader(bytes.NewReader(buff.Bytes()))
		if err != nil {
			log.Println(err)
		}
		defer rdr.Close()
		scanner := bufio.NewScanner(rdr)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			// Avoid filling up memory too much
			if len(dataPipe) > 50000 {
				time.Sleep(500 * time.Millisecond)
			}
			dataPipe <- processLine(filePattern, scanner.Text())
		}
	}
	s3wg.Done()
}

// processLocalFile reads a file and process each of the lines and sends them to the
// given open channel
func processLocalFile(path string, dataPipe chan *accessLogEntry) {
	inFile, err := os.Open(path)
	if err != nil {
		log.Printf("Error while reading file %s: %s\n", path, err)
	}
	defer inFile.Close()
	rdr, err := gzip.NewReader(inFile)
	if err != nil {
		log.Println(err)
	}
	defer rdr.Close()
	scanner := bufio.NewScanner(rdr)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		// Avoid filling up memory too much
		if len(dataPipe) > 50000 {
			time.Sleep(500 * time.Millisecond)
		}
		dataPipe <- processLine(filePattern, scanner.Text())
	}
}

// dbCreateTable creates the table if it does not exists
func dbCreateTable(db *sql.DB, tableName string) {
	crStmt, err := db.Prepare(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (`year` INT(4), `month` INT(2), `day` INT(2), `hour` INT(2), `bytes` BIGINT, `hlsVersion` VARCHAR(8), `bitrate` VARCHAR(30), `responseCode` VARCHAR(4), `userAgent` VARCHAR(512))", tableName))
	if err != nil {
		log.Println(err)
	}

	if _, err = crStmt.Exec(); err != nil {
		log.Println(err)
	}
	if err = crStmt.Close(); err != nil {
		log.Println(err)
	}
}

// dbInsertElt adds an accesslog entry to the table
func dbInsertElt(stmt *sql.Stmt, elem *accessLogEntry) {
	var err error

	// sanity
	agentLen := len(elem.userAgent)
	if agentLen > 511 {
		agentLen = 511
	}
	if _, err = stmt.Exec(elem.year, elem.month, elem.day, elem.hour, elem.bytes, elem.hlsVersion, elem.bitrate, elem.responseCode, elem.userAgent); err != nil {
		log.Println(err)
	}

}

// dbCheckForCommit commits the transaction if idx is over maxIdx and resets idx to 0
func dbCheckForCommit(idx *int, maxIdx int, stmt *sql.Stmt, tx *sql.Tx) {
	if *idx > maxIdx {
		var err error
		if err = stmt.Close(); err != nil {
			log.Println(err)
		}
		if err = tx.Commit(); err != nil {
			log.Println(err)
		}
		*idx = 0
	}
}

// Takes the data out of the given channel and pushes it to the given mysql
// table
func channelToDB(user, pwd, host, database, tableName string, dataPipe chan *accessLogEntry) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@%s/%s?charset=utf8", user, pwd, host, database))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	dbCreateTable(db, tableName)

	var (
		tx   *sql.Tx
		stmt *sql.Stmt
	)
	flagIdx := 0
	for elem := range dataPipe {

		if elem == nil {
			continue
		}

		// Prepared statement in the transaction has to be re-prepared every time we
		// commit as it closes the transaction
		if flagIdx == 0 {
			tx, err = db.Begin()
			if err != nil {
				log.Println(err)
			}
			stmt, err = tx.Prepare(fmt.Sprintf("insert into `%s` (`year`, `month`, `day`, `hour`, `bytes`, `hlsVersion`, `bitrate`, `responseCode`, `userAgent`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", tableName))
			if err != nil {
				log.Println(err)
			}
		}

		dbInsertElt(stmt, elem)

		flagIdx++
		dbCheckForCommit(&flagIdx, 10000, stmt, tx)
	}
	dbCheckForCommit(&flagIdx, 0, stmt, tx)

	wg.Done()
}

// getLocalFiles returns the regular files available in the given directory and its subdirectories
// If the given path is a regular file, it returns the file
// If the given path is a non-regular file (a mode type bit is set), returns an empty array
func getLocalFiles(path string) []*string {
	result := []*string{}
	var files []os.FileInfo

	rootF, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}

	if rootF.IsDir() {
		f, err := os.OpenFile(path, os.O_RDONLY, 0500)
		if err != nil {
			log.Fatal(err)
		}
		if files, err = f.Readdir(0); err != nil {
			log.Fatal(err)
		}
	} else {
		if rootF.Mode().IsRegular() {
			result = append(result, &path)
		}
	}

	for _, file := range files {
		sub := filepath.Join(path, file.Name())
		// discarding special files
		switch mode := file.Mode(); {
		case mode.IsDir():
			subres := getLocalFiles(sub)
			result = append(result, subres...)
		case mode.IsRegular():
			result = append(result, &sub)
		}
	}
	return result
}

func dbQueryToCSV(db *sql.DB, query string, csvWriter *csv.Writer) error {
	rows, err := db.Query(query)
	if err != nil {
		return err
	}

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	if err = csvWriter.Write(columns); err != nil {
		return err
	}

	// Make a slice for the values
	count := len(columns)
	values := make([]interface{}, count)
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// Fetch rows
	for rows.Next() {
		row := make([]string, count)
		if err = rows.Scan(scanArgs...); err != nil {
			return err
		}

		for i := range columns {
			var value interface{}
			rawValue := values[i]

			byteArray, ok := rawValue.([]byte)
			if ok {
				value = string(byteArray)
			} else {
				value = rawValue
			}
			row[i] = fmt.Sprintf("%v", value)
		}

		if err = csvWriter.Write(row); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return rows.Err()
}

// generateReport generates a standard report in a summary file
func generateReport(user, pwd, host, database, tableName, reportPath string) {
	if len(reportPath) == 0 {
		log.Println("-report-path flag empty. Skipping report generation")
		return
	}

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@%s/%s?charset=utf8", user, pwd, host, database))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	f, errF := os.Create(reportPath)
	if errF != nil {
		log.Fatal(err)
	}
	defer f.Close()

	csvWriter := csv.NewWriter(f)
	queries := []struct {
		title, query string
	}{
		{"Requests per day", "select CONCAT(year, '-', month, '-', day) as date, count(*) as nbrcalls from `" + tableName + "` where responseCode='200' and userAgent not like 'Pingdom%' and userAgent != 'ZmEu' group by year, month, day order by year, month, day, nbrcalls"},
		{"Requests per day per user agent ", "select CONCAT(year, '-', month, '-', day) as date, userAgent, count(*) as nbrcalls from `" + tableName + "` where responseCode='200' and userAgent not like 'Pingdom%' and userAgent != 'ZmEu' group by year, month, day, userAgent order by year, month, day, userAgent, nbrcalls"},
		{"Bytes by bitrate", "select bitrate, sum(bytes) as total_bytes, count(*) as nbrcalls from `" + tableName + "` where responseCode='200' and userAgent not like 'Pingdom%' and userAgent != 'ZmEu' group by bitrate order by total_bytes desc"},
		{"Bytes by user agent", "select userAgent, sum(bytes) as total_bytes, count(*) as nbrcalls from `" + tableName + "` where responseCode='200' and userAgent not like 'Pingdom%' and userAgent != 'ZmEu' group by userAgent order by total_bytes desc"},
	}
	for _, q := range queries {
		if err = csvWriter.Write([]string{q.title}); err != nil {
			log.Fatal(err)
		}
		if err = dbQueryToCSV(db, q.query, csvWriter); err != nil {
			log.Fatal(err)
		}
		if err = csvWriter.Write(nil); err != nil {
			log.Fatal(err)
		}
		csvWriter.Flush()
	}
}

func main() {
	var (
		fPath, dbName, dbHost, dbUser, dbPassword, dbTable, reportFile, s3Bucket, s3Path string
		recursive                                                                        bool
	)
	flag.BoolVar(&recursive, "recursive", false, "Considers the -file-path input as directory and will search for files to process inside. Environment variable: RECURSIVE")
	flag.StringVar(&fPath, "file-path", "", "Path to the log file. If -recursive flag is set, this is considered as a directory. Environment variable: FILE_PATH")
	flag.StringVar(&dbName, "db-name", "accesslogs", "Name of the DB to connect to. Environment variable: DB_NAME")
	flag.StringVar(&dbHost, "db-host", "", "Name of the DB server to connect to. Environment variable: DB_HOST")
	flag.StringVar(&dbUser, "db-user", "", "User name to use to connect to the DB. Environment variable: DB_USER")
	flag.StringVar(&dbPassword, "db-pwd", "", "Password to use to connect to the DB. Environment variable: DB_PWD")
	flag.StringVar(&dbTable, "db-table", "", "Name of the table to import the data in. Environment variable: DB_TABLE")
	flag.StringVar(&reportFile, "report-path", "", "Path of the standard report summary you want to generate. If left empty, the report won't be generated. Environment variable: REPORT_PATH")
	flag.StringVar(&s3Bucket, "s3-bucket", "", "Name of the bucket where your access logs are stored. Incompatible with -file-path. Only specify it if you want to read your access logs directly from s3. Environment variable: S3_BUCKET")
	flag.StringVar(&s3Path, "s3-path", "", "Path in the s3 bucket where the access logs are stored. Important: -recursive is not needed for s3. The script will look for all the files in the directory if the provided s3-path is a folder. Environment variable: S3_PATH")
	envflag.Parse()

	dp := make(chan *accessLogEntry)
	wg.Add(1)
	go channelToDB(dbUser, dbPassword, dbHost, dbName, dbTable, dp)

	if len(fPath) > 0 {
		fInput := []*string{&fPath}
		if recursive {
			fInput = getLocalFiles(fPath)
		}
		for _, f := range fInput {
			log.Printf("Processing file %s\n", *f)
			processLocalFile(*f, dp)
		}
	}
	if len(s3Bucket) > 0 {
		processS3Files(s3Bucket, s3Path, dp)
	}
	close(dp)
	wg.Wait()
	log.Printf("Generating report")
	generateReport(dbUser, dbPassword, dbHost, dbName, dbTable, reportFile)
}
