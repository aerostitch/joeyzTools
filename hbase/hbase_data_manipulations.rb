#!/usr/local/hbase/bin/hbase org.jruby.Main
# ************************************************************************
#
# Author: Joseph Herlant
# Last updated: 2014-09-01
# Description:
#  This script is intended to play a little bit with data manipulation in
#  Apache HBase by:
#   - Creating a 'songs' table (dropping it first if exists)
#   - Put some data in it
#   - Listing the available tables
#   - Listing the available data in the tested table
#
# Tested on: a CDH3U6 instance of HBase installed in /usr/local/hbase
#
# ************************************************************************

require 'java'
# General HBase classes
import 'org.apache.hadoop.hbase.HBaseConfiguration'
import 'org.apache.hadoop.hbase.HColumnDescriptor'
import 'org.apache.hadoop.hbase.HTableDescriptor'
# HBase client classes
import 'org.apache.hadoop.hbase.client.HBaseAdmin'
import 'org.apache.hadoop.hbase.client.HTable'
import 'org.apache.hadoop.hbase.client.Put'
import 'org.apache.hadoop.hbase.client.Scan'
import 'org.apache.hadoop.hbase.util.Bytes'
# Used to disable hbase logging debug level
import 'org.apache.log4j.Logger'

ZOOKEEPERSERVER='localhost'
ZOOKEEPERPORT='2181'
TESTTABLENAME='pink_floyd_songs'
TESTCOLFAMILY='attributes'

# Converts every given arguments to Java bytes
# @param args [String] the input arguments you want to convert.
# @return [bytes[]] the given arguments converted to the expected format.
def jbytes( *args )
  args.map { |arg| arg.to_s.to_java_bytes }
end

# Prepare a `org.apache.hadoop.hbase.client.Put` object with a song
#  and its given attributes.
# @param title [String] The title of the song (key of the row.
# @param attributes [Hash] The list of columns qualifiers (keys)
#  and their values (the value)
# @return [org.apache.hadoop.hbase.client.Put] The row to put.
def prep_song( title, attributes = {} )
  p = Put.new( *jbytes( title ) )
  attributes.each do |qualifier, value|
    p.add( *jbytes( TESTCOLFAMILY, qualifier, value ) )
  end
  return p
end

# Just forcing logging level for the client
Logger.getLogger("org.apache.zookeeper").setLevel(org.apache.log4j.Level::WARN)
Logger.getLogger("org.apache.hadoop.hbase.client").setLevel(org.apache.log4j.Level::WARN)
Logger.getLogger("org.apache.hadoop.hbase.catalog").setLevel(org.apache.log4j.Level::WARN)

conf = HBaseConfiguration.create()
conf.set( 'hbase.zookeeper.quorum', ZOOKEEPERSERVER )
conf.set( 'hbase.zookeeper.property.clientPort', ZOOKEEPERPORT )
admin = HBaseAdmin.new( conf )

# Preparing the new table structure
desc = HTableDescriptor.new( TESTTABLENAME )
desc.addFamily( HColumnDescriptor.new( TESTCOLFAMILY ) )

# Deleting table if already exists and recreating it
if admin.tableExists( TESTTABLENAME )
  admin.disableTable( TESTTABLENAME )
  admin.deleteTable( TESTTABLENAME )
end
admin.createTable( desc )

# Adding some data to it...
table = HTable.new( conf, TESTTABLENAME )
s = prep_song( "Astronomy Domine", { "album" => "The Piper at the Gates of Dawn", "duration" => '04:12' } )
table.put( s )
s = prep_song( "Bike", { "album" => "The Piper at the Gates of Dawn", "duration" => '03:23' } )
table.put( s )
s = prep_song( "Money", { "album" => "The Dark Side of the Moon", "rating" => 5 } )
table.put( s )

# Making a little printing
puts '*' * 25 + " List of available tables:"
admin.listTables.each{ |tbl| puts " * #{tbl.nameAsString}" }

puts '*' * 25 + " Available Pink Floyd songs:"
scan = Scan.new
# scan.addFamily( *jbytes( TESTCOLFAMILY  ) )
scanner = table.getScanner( scan )
begin
  while row = scanner.next do
    title = Bytes.toString( row.getRow )
    puts " * Song \"#{title}\" has the following registered cells:"
    # Listing all the cells available for the row
    row.list.each do |l|
      fam = Bytes.toString( l.family )
      qual = Bytes.toString( l.qualifier )
      val = Bytes.toString( l.value )
      puts "      -> \"#{val}\" for \"#{fam}:#{qual}\""
    end
    # album = Bytes.toString(row.getValue( *jbytes( TESTCOLFAMILY, "album")))
    # puts " * #{title} (from \"#{album}\")"
  end
ensure
    scanner.close
end
table.close

