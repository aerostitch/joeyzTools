package main

/*
This tool analyzes access logs for classic ELB access logs and pushes them
inside a MySQL/MariaDB database so that you can generate some reporting on it.

Easy way to get a DB:
```
docker run --name some-mariadb -e MYSQL_ROOT_PASSWORD=my-secret-pw -e MYSQL_DATABASE=accesslogs -p 3306:3306 -d mariadb:latest
```

To bulk load your files in this case:
```
DB_NAME=accesslogs
TBL=bla
go run aws_elb_log_analyzer.go  -db-host "tcp(172.17.0.2)" \
																-db-create-table \
																-db-name ${DB_NAME} \
																-db-user root \
																-db-pwd my-secret-pw \
																-db-table ${TBL} \
																-recursive \
																-file-path /tmp/my_local_access_logs/
```

Custom reports examples based on the imported data (here we exclude the calls from Pingdom and stuffs that we now are script kiddies playing around):
 * By day and IP: `select year, month, day, sourceIP, count(*) as nbrcalls from bla group by year, month, day, sourceIP order by nbrcalls;`
 * By uri: `select SUBSTRING_INDEX(uri, '?', 1), count(*) as nbrcalls from bla group by SUBSTRING_INDEX(uri, '?', 1) order by nbrcalls;`
 * By userAgent: `select SUBSTRING_INDEX(userAgent, ' (', 1), count(*) as nbrcalls from bla group by SUBSTRING_INDEX(userAgent, ' (', 1) order by nbrcalls;`
 * A bit of filtering: `select year, month, day, hour, SUBSTRING_INDEX(userAgent, ' (', 1) as agent, SUBSTRING_INDEX(uri, '?', 1) as uri, count(*) as nbrcalls from bla where userAgent not like 'Pingdom%' and userAgent != 'ZmEu' group by year, month, day, hour, SUBSTRING_INDEX(userAgent, ' (', 1), SUBSTRING_INDEX(uri, '?', 1) order by year, month, day, hour, nbrcalls;`

Usage example in a script:
```
#!/bin/bash
if [ $# -ne 1 ]; then
  echo "argument required"
  exit 1
fi

TBL=$1
BUCKET=my-elb-logs
aws s3 cp --recursive --exclude "*" --include "*2018/*" s3://${BUCKET}/${TBL}/AWSLogs/ /tmp/${TBL}
find /tmp/${TBL} -type f -name '*.log' -o -name '*.txt' | while read f; do
  echo "Processing $f"
  go run aws_elb_log_analyzer.go -db-create-table -db-host "tcp(172.17.0.2)" -db-name accesslogs -db-user root -db-pwd my-secret-pw -db-table ${TBL} -file-path $f
done
mysql -h 172.17.0.2 -u root --password=my-secret-pw --database accesslogs -e "select CONCAT(year, '-', month, '-', day) as date, SUBSTRING_INDEX(userAgent, ' ', 1) as agent, SUBSTRING_INDEX(SUBSTRING_INDEX(REPLACE(uri,'//','/'), '?', 1), '/', 3) as shorturi, count(*) as nbrcalls from \`${TBL}\` where userAgent not like 'Pingdom%' and userAgent != 'ZmEu' group by year, month, day,  SUBSTRING_INDEX(userAgent, ' ', 1), SUBSTRING_INDEX(SUBSTRING_INDEX(REPLACE(uri,'//','/'), '?', 1), '/', 3) order by year, month, day, nbrcalls" -B > /tmp/${TBL}_short.tsv

echo "Requests per day" > /tmp/${TBL}_summary.tsv
mysql -h 172.17.0.2 -u root --password=my-secret-pw --database accesslogs -e "select CONCAT(year, '-', month, '-', day) as date, count(*) as nbrcalls from \`${TBL}\` where userAgent not like 'Pingdom%' and userAgent != 'ZmEu' group by year, month, day order by year, month, day, nbrcalls" -B >> /tmp/${TBL}_summary.tsv
echo "" >> /tmp/${TBL}_summary.tsv
echo "Requests per method and scheme" >> /tmp/${TBL}_summary.tsv
mysql -h 172.17.0.2 -u root --password=my-secret-pw --database accesslogs -e "select method, scheme, count(*) as nbrcalls from \`${TBL}\` where userAgent not like 'Pingdom%' and userAgent != 'ZmEu' group by method, scheme order by nbrcalls" -B >> /tmp/${TBL}_summary.tsv
echo "" >> /tmp/${TBL}_summary.tsv
echo "Top 10 source IP" >> /tmp/${TBL}_summary.tsv
mysql -h 172.17.0.2 -u root --password=my-secret-pw --database accesslogs -e "select * from (select sourceIP, count(*) as nbrcalls from \`${TBL}\` where userAgent not like 'Pingdom%' and userAgent != 'ZmEu' group by sourceIP order by nbrcalls desc) t limit 10;" -B >> /tmp/${TBL}_summary.tsv
echo "" >> /tmp/${TBL}_summary.tsv
echo "Top 10 full user agent" >> /tmp/${TBL}_summary.tsv
mysql -h 172.17.0.2 -u root --password=my-secret-pw --database accesslogs -e "select * from (select userAgent, count(*) as nbrcalls from \`${TBL}\` where userAgent not like 'Pingdom%' and userAgent != 'ZmEu' group by userAgent order by nbrcalls desc) t limit 10;" -B >> /tmp/${TBL}_summary.tsv
echo "" >> /tmp/${TBL}_summary.tsv
echo "Top 10 short user agent" >> /tmp/${TBL}_summary.tsv
mysql -h 172.17.0.2 -u root --password=my-secret-pw --database accesslogs -e "select * from (select SUBSTRING_INDEX(SUBSTRING_INDEX(userAgent, ' ', 1),'(',1) as userAgent, count(*) as nbrcalls from \`${TBL}\` where userAgent not like 'Pingdom%' and userAgent != 'ZmEu' group by SUBSTRING_INDEX(SUBSTRING_INDEX(userAgent, ' ', 1),'(',1) order by nbrcalls desc) t limit 10;" -B >> /tmp/${TBL}_summary.tsv
echo "" >> /tmp/${TBL}_summary.tsv
echo "Top 10 root uri path" >> /tmp/${TBL}_summary.tsv
mysql -h 172.17.0.2 -u root --password=my-secret-pw --database accesslogs -e "select * from (select SUBSTRING_INDEX(SUBSTRING_INDEX(REPLACE(uri,'//','/'), '?', 1), '/', 2) as root_uri, count(*) as nbrcalls from \`${TBL}\` where userAgent not like 'Pingdom%' and userAgent != 'ZmEu' group by SUBSTRING_INDEX(SUBSTRING_INDEX(REPLACE(uri,'//','/'), '?', 1), '/', 2) order by nbrcalls desc) t limit 10;" -B >> /tmp/${TBL}_summary.tsv
echo "" >> /tmp/${TBL}_summary.tsv
echo "Top 10 short uri path" >> /tmp/${TBL}_summary.tsv
mysql -h 172.17.0.2 -u root --password=my-secret-pw --database accesslogs -e "select * from (select SUBSTRING_INDEX(SUBSTRING_INDEX(REPLACE(uri,'//','/'), '?', 1), '/', 3) as short_uri, count(*) as nbrcalls from \`${TBL}\` where userAgent not like 'Pingdom%' and userAgent != 'ZmEu' group by SUBSTRING_INDEX(SUBSTRING_INDEX(REPLACE(uri,'//','/'), '?', 1), '/', 3) order by nbrcalls desc) t limit 10;" -B >> /tmp/${TBL}_summary.tsv
echo "" >> /tmp/${TBL}_summary.tsv
echo "Top 10 raw uri path" >> /tmp/${TBL}_summary.tsv
mysql -h 172.17.0.2 -u root --password=my-secret-pw --database accesslogs -e "select * from (select SUBSTRING_INDEX(uri,'?', 1) as uri, count(*) as nbrcalls from \`${TBL}\` where userAgent not like 'Pingdom%' and userAgent != 'ZmEu' group by SUBSTRING_INDEX(uri, '?', 1) order by nbrcalls desc) t limit 10;" -B >> /tmp/${TBL}_summary.tsv
rm -rf /tmp/${TBL}/
```

*/

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gobike/envflag"
)

var wg sync.WaitGroup
var classicELBPattern = regexp.MustCompile(`^([^ ]*) ([^ ]*) ([^ ]*):([0-9]*) ([^ ]*)[:\-]([0-9]*) ([-.0-9]*) ([-.0-9]*) ([-.0-9]*) (|[-0-9]*) (-|[-0-9]*) ([-0-9]*) ([-0-9]*) "([^ ]*) ([^ ]*) (- |[^ ]*)" "([^"]*)" ([A-Z0-9-]+) ([A-Za-z0-9.-]*)$`)

type accessLogEntry struct {
	year, month, day, hour                           int
	sourceIP, method, domain, scheme, uri, userAgent string
}

// processLine takes a line and the compiled regex and returns a accessLogEntry
func processLine(re *regexp.Regexp, line string) *accessLogEntry {
	entry := accessLogEntry{}

	result := re.FindStringSubmatch(line)

	// do not process incorrect lines
	if len(result) < 18 {
		return nil
	}
	layout := "2006-01-02T15:04:05.000000Z"
	mDate, err := time.Parse(layout, result[1])
	if err != nil {
		log.Println(err)
	}
	entry.year = mDate.Year()
	entry.month = int(mDate.Month())
	entry.day = mDate.Day()
	entry.hour = mDate.Hour()

	entry.sourceIP = result[3]
	entry.method = result[14]

	u, err := url.Parse(result[15])
	if err != nil {
		log.Println(err)
	} else {
		entry.domain = u.Hostname()
		entry.scheme = u.Scheme
		entry.uri = u.RequestURI()
	}

	entry.userAgent = result[17]
	return &entry
}

// processLocalFile reads a file and process each of the lines and sends them to the
// given open channel
func processLocalFile(path string, dataPipe chan *accessLogEntry) {
	inFile, err := os.Open(path)
	if err != nil {
		log.Printf("Error while reading file %s: %s\n", path, err)
	}
	defer inFile.Close()
	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		dataPipe <- processLine(classicELBPattern, scanner.Text())
	}
}

// dbCreateTable creates the table if it does not exists
func dbCreateTable(db *sql.DB, tableName string) {
	crStmt, err := db.Prepare(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (`year` INT(4), `month` INT(2), `day` INT(2), `hour` INT(2), `sourceIP` VARCHAR(128), `method` VARCHAR(8), `domain` VARCHAR(256), `scheme` VARCHAR(8), `uri` VARCHAR(512), `userAgent` VARCHAR(512))", tableName))
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

// insertElt adds an accesslog entry to the table
func insertElt(db *sql.DB, stmt *sql.Stmt, tx *sql.Tx, elem *accessLogEntry, tableName *string, flagIdx *int) {
	var err error
	// Prepared statement in the transaction has to be re-prepared every time we
	// commit as it closes the transaction
	if *flagIdx == 0 {
		tx, err = db.Begin()
		if err != nil {
			log.Println(err)
		}
		stmt, err = tx.Prepare(fmt.Sprintf("insert into `%s` (`year`, `month`, `day`, `hour`, `sourceIP`, `method`, `domain`, `scheme`, `uri`, `userAgent`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", *tableName))
		if err != nil {
			log.Println(err)
		}
		defer stmt.Close()
	}

	// sanity
	uriLen := len(elem.uri)
	if uriLen > 511 {
		uriLen = 511
	}
	agentLen := len(elem.userAgent)
	if agentLen > 511 {
		agentLen = 511
	}
	if _, err = stmt.Exec(elem.year, elem.month, elem.day, elem.hour, elem.sourceIP, elem.method, elem.domain, elem.scheme, elem.uri[:uriLen], elem.userAgent[:agentLen]); err != nil {
		log.Println(err)
	}

	(*flagIdx)++
	if *flagIdx > 10000 {
		if err = tx.Commit(); err != nil {
			log.Println(err)
		}
		*flagIdx = 0
	}
}

// Takes the data out of the given channel and pushes it to the given mysql
// table
func channelToDB(user, pwd, host, database, tableName string, createTbl bool, dataPipe chan *accessLogEntry) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@%s/%s?charset=utf8", user, pwd, host, database))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if createTbl {
		dbCreateTable(db, tableName)
	}

	var (
		tx   *sql.Tx
		stmt *sql.Stmt
	)
	flagIdx := 0
	for elem := range dataPipe {

		if elem == nil {
			continue
		}

		insertElt(db, stmt, tx, elem, &tableName, &flagIdx)

	}
	if flagIdx != 0 {
		if err = tx.Commit(); err != nil {
			log.Println(err)
		}
	}

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

func main() {
	var (
		fPath, dbName, dbHost, dbUser, dbPassword, dbTable string
		dbCreateTable, recursive                           bool
	)
	flag.BoolVar(&recursive, "recursive", false, "Considers the -file-path input as directory and will search for files to process inside. Environment variable: RECURSIVE")
	flag.StringVar(&fPath, "file-path", "text", "Path to the log file. If -recursive flag is set, this is considered as a directory. Environment variable: FILE_PATH")
	flag.StringVar(&dbName, "db-name", "accesslogs", "Name of the DB to connect to. Environment variable: DB_NAME")
	flag.StringVar(&dbHost, "db-host", "", "Name of the DB server to connect to. Environment variable: DB_HOST")
	flag.StringVar(&dbUser, "db-user", "", "User name to use to connect to the DB. Environment variable: DB_USER")
	flag.StringVar(&dbPassword, "db-pwd", "", "Password to use to connect to the DB. Environment variable: DB_PWD")
	flag.StringVar(&dbTable, "db-table", "", "Name of the table to import the data in. Environment variable: DB_TABLE")
	flag.BoolVar(&dbCreateTable, "db-create-table", false, "Whether to create the table if it does not exists. Environment variable: DB_CREATE_TABLE")
	envflag.Parse()

	dp := make(chan *accessLogEntry)
	wg.Add(1)
	go channelToDB(dbUser, dbPassword, dbHost, dbName, dbTable, dbCreateTable, dp)

	// TODO: process s3 file & folder
	if len(fPath) > 0 {
		fInput := []*string{&fPath}
		if recursive {
			fInput = getLocalFiles(fPath)
		}
		for _, f := range fInput {
			processLocalFile(*f, dp)
		}
	}
	close(dp)
	wg.Wait()
}
