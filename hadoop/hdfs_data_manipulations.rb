#!/usr/local/hbase/bin/hbase org.jruby.Main
# ************************************************************************
#
# Author: Joseph Herlant
# Last updated: 2014-09-01
# Description:
#  This script is intended to play a little bit with data manipulation in
#  Apache Hadoop HDFS filesystem by:
#   - Listing files (and permissions) in user's home directory if it exists
#   - Creating the home directory if it does not exists
#   - Displaying the content of the copied file
#
# Tested on: a CDH3U6 instance of Hadoop installed in /usr/local/hbase
#
# ************************************************************************

require 'java'
import 'org.apache.hadoop.conf.Configuration'
import 'org.apache.hadoop.fs.FileSystem'
import 'org.apache.hadoop.fs.Path'
import 'org.apache.hadoop.fs.permission.FsPermission'

HDFSNAMENODE = 'localhost'
HDFSPORT     =  12345
LOCAL_DIR    = '/tmp'
LOCAL_FILE   = 'test_file'
local_path   = File.join( LOCAL_DIR, LOCAL_FILE )

abort("\e[0;31m[ERROR] File #{local_path} not found.\e[m") unless File.exists? local_path

conf = Configuration.new
# conf.each { |c| puts c }
# puts conf.get( 'fs.default.name' )
conf.set( 'fs.default.name', "hdfs://#{HDFSNAMENODE}:#{HDFSPORT}" )

fs = FileSystem.get( conf )
srcfile = Path.new( local_path )
homedir = fs.get_home_directory

# Listing the home directory if exists
if fs.is_directory? homedir
  puts '*' * 25 + " Files in your home directory are:"
  fs.list_status( homedir ).each { |f| puts " * #{f.get_owner}:#{f.get_group} #{f.get_permission} #{f.get_path}" }
else
  puts "\e[0;33m[WARNING] Your HDFS home directory #{fs.get_home_directory.to_s} does not exist.\e[m"
end

# Copying the chosen file to your home directory
begin
  fs.mkdirs homedir unless fs.is_directory? homedir
  fs.copy_from_local_file( srcfile, homedir )
rescue NativeException
  puts "\e[0;31m[ERROR] Unable to copy local file to your hdfs home directory.\e[m"
  puts "The error was:"
  abort("#{$!}")
end

# Displaying content of the uploaded file
puts '*' * 25 + " Content of the copied file:"
hdfs_file_path = File.join( homedir.to_s, LOCAL_FILE )
file = fs.open( Path.new( hdfs_file_path ) )
while ( l = file.read_line )
  puts l
end
file.close

