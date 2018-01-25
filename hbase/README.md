# HBase-related scripts

## hbase_data_manipulations.rb

### Description

This script is intended to play a little bit with data manipulation in
Apache HBase using the JRuby engine embedded inside HBase by:
 - Creating a 'songs' table (dropping it first if exists)
 - Put some data in it
 - Listing the available tables
 - Listing the available data in the tested table

**Tested on:** a CDH3U6 instance of HBase installed in /usr/local/hbase

**Note:** This script was written to be used as a basis on a really old version
of HBase. It will probably not work with newer versions of HBase.

## hbase_delete_data.rb

### Description

This script is deleting data based on a pattern from a given table using the
JRuby engine embedded inside HBase.

This script deleting rows in hbase 1 by 1 (not to impact production).
No, this is not efficient and it was not meant to be working with recent
versions of HBase. The problematic was to make it work on a CDH3U5 (0.90).

**Tested on:** a CDH3U5 instance of HBase installed in /usr/local/hbase

**Notes:**
* This script was written to be used as a basis on a really old version
of HBase. It will probably not work with newer versions of HBase.
* Sometimes when you are running low in memory, you can try: `export _JAVA_OPTIONS="-Xms128M -Xmx256M -XX:NewSize=100M"`

### Usage

Provided that your hbase installation is in `/usr/local/hbase/`:

```
sudo -H -E -u hadoop /usr/local/hbase/bin/hbase org.jruby.Main hbase_delete_data.rb [options]
    -z --zookeeper ZOOKEEPER_HOSTNAME            Zookeeper hostname or IP - default: localhost
    -p, --zk-port PORT               Zookeeper port - default: 2181
    -t, --table TABLENAME            Table to delete records from - default: my_big_table
    -d, --delete-pattern REGEX       Pattern of the keys you want to delete - default: "yyyymmddhh=201[0-5]\d+$"
    -f, --user-rowfilter             Use the built-in RowFilter (uses less network but can pressure the regionservers) - default: false
    -s, --non-suspect-pattern REGEX  Pattern to list all the non-delted elements NOT matching a given pattern. Empty pattern disables that. Use that to detect suspect keys - default: ""
    -k, --delete-suspect-keys         Deletes the keys detected to be suspect from the "--non-suspect-pattern" option. - default: false
```

## simple_hbase_manip.sh

A script that does several modifications on HDFS and then on HBase using shell.

