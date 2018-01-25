# Hadoop-related scripts

## hdfs_data_manipulations.rb

This script is intended to play a little bit with data manipulation in
Apache Hadoop HDFS filesystem using the JRuby engine embedded inside HBase by:
 - Listing files (and permissions) in user's home directory if it exists
 - Creating the home directory if it does not exists
 - Displaying the content of the copied file

**Tested on:** a CDH3U6 instance of Hadoop installed in /usr/local/hbase

**Note:** This script was written to be used as a basis on a really old version
of Hadoop/HBase. It will probably not work with newer versions of Hadoop/HBase.

