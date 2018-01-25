#!/usr/local/hbase/bin/hbase org.jruby.Main

# ************************************************************************
#
# Author: Joseph Herlant
# Last updated: 2016-09-29
# Description:
#  This script is deleting data based on a pattern from a given table.
#  This script deleting rows in hbase 1 by 1 (not to impact production).
#  No, this is not efficient and it was not meant to be working with recent
#  versions of HBase. The problematic was to make it work on a CDH3U5 (0.90).
#
# Tested on: a CDH3U5 instance of HBase installed in /usr/local/hbase
# To start it:
# sudo -H -E -u hadoop /usr/local/hbase/bin/hbase org.jruby.Main /tmp/hbase_delete_data.rb
#
# Usage: sudo -H -E -u hadoop /usr/local/hbase/bin/hbase org.jruby.Main hbase_delete_data.rb [options]
#     -z --zookeeper ZOOKEEPER_HOSTNAME            Zookeeper hostname or IP - default: localhost
#     -p, --zk-port PORT               Zookeeper port - default: 2181
#     -t, --table TABLENAME            Table to delete records from - default: my_big_table
#     -d, --delete-pattern REGEX       Pattern of the keys you want to delete - default: "yyyymmddhh=201[0-5]\d+$"
#     -f, --user-rowfilter             Use the built-in RowFilter (uses less network but can pressure the regionservers) - default: false
#     -s, --non-suspect-pattern REGEX  Pattern to list all the non-delted elements NOT matching a given pattern. Empty pattern disables that. Use that to detect suspect keys - default: ""
#     -k, --delete-suspect-keys         Deletes the keys detected to be suspect from the "--non-suspect-pattern" option. - default: false
#     -v, --verbose                     Displays more verbose output. - default: false
#
# Sometimes when you are running low in memory, you can try: export _JAVA_OPTIONS="-Xms128M -Xmx256M -XX:NewSize=100M"
#
# ************************************************************************

require 'optparse'
require 'java'

# General HBase classes
import 'org.apache.hadoop.hbase.HBaseConfiguration'
# HBase client classes
import 'org.apache.hadoop.hbase.client.HTable'
import 'org.apache.hadoop.hbase.client.Scan'
import 'org.apache.hadoop.hbase.client.Delete'
import 'org.apache.hadoop.hbase.util.Bytes'
import 'org.apache.hadoop.hbase.filter.CompareFilter'
import 'org.apache.hadoop.hbase.filter.RowFilter'
import 'org.apache.hadoop.hbase.filter.RegexStringComparator'
import 'org.apache.hadoop.hbase.filter.FirstKeyOnlyFilter'
import 'org.apache.hadoop.hbase.filter.KeyOnlyFilter'
import 'org.apache.hadoop.hbase.filter.FilterList'
# Used to disable hbase logging debug level
import 'org.apache.log4j.Logger'

# Converts every given arguments to Java bytes
# @param args [String] the input arguments you want to convert.
# @return [bytes[]] the given arguments converted to the expected format.
def jbytes( *args )
  args.map { |arg| arg.to_s.to_java_bytes }
end

# Just forcing logging level for the client
Logger.getLogger("org.apache.zookeeper").setLevel(org.apache.log4j.Level::WARN)
Logger.getLogger("org.apache.hadoop.hbase.client").setLevel(org.apache.log4j.Level::WARN)
Logger.getLogger("org.apache.hadoop.hbase.catalog").setLevel(org.apache.log4j.Level::WARN)

options = {
  :zookeeper => 'localhost',
  :zkport => 2181,
  :tablename => 'my_big_table',
  :delete_pattern => 'yyyymmddhh=201[0-5]\d+$',
  :use_rowfilter => false,
  :suspects_antipattern => '',
  :delete_suspects => false,
  :verbose => false,
}
OptionParser.new do |opts|
  opts.banner = "Usage: hbase_delete_data.rb [options]"

  opts.on('-z', '--zookeeper ZOOKEEPER_HOSTNAME', 'Zookeeper hostname or IP - default: localhost') { |v| options[:zookeeper] = v }
  opts.on('-p', '--zk-port PORT', Integer, 'Zookeeper port - default: 2181') { |v| options[:zkport] = v }
  opts.on('-t', '--table TABLENAME', 'Table to delete records from - default: my_big_table') { |v| options[:tablename] = v }
  opts.on('-d', '--delete-pattern REGEX', 'Pattern of the keys you want to delete - default: "yyyymmddhh=201[0-5]\d+$"') { |v| options[:delete_pattern] = v }
  opts.on('-f', '--user-rowfilter', 'Use the built-in RowFilter (uses less network but can pressure the regionservers) - default: false') { |v| options[:use_rowfilter] = v }
  opts.on('-s', '--non-suspect-pattern REGEX', 'Pattern to list all the non-delted elements NOT matching a given pattern. Empty pattern disables that. Use that to detect suspect keys - default: ""') { |v| options[:suspects_antipattern] = v }
  opts.on('-k', '--delete-suspect-keys', 'Deletes the keys detected to be suspect from the "--non-suspect-pattern" option. - default: false') { |v| options[:delete_suspects] = v }
  opts.on('-v', '--verbose', 'Displays more verbose output. - default: false') { |v| options[:verbose] = v }
end.parse!

puts "Using the following options:"
options.map{|k,v| puts "  #{k} = #{v}" }

conf = HBaseConfiguration.create()
conf.set('hbase.zookeeper.quorum', options[:zookeeper])
conf.set('hbase.zookeeper.property.clientPort', options[:zkport].to_s)
conf.set('hbase.client.scanner.caching', '50000') # prefetch size
conf.set('hbase.client.scanner.lease.timeout', (60 * 60 * 1000).to_s)
conf.set('hbase.rpc.timeout', (60 * 60 * 1000).to_s)
conf.set('hbase.regionserver.lease.period', (60 * 60 * 1000).to_s)

# Adding some data to it...
table = HTable.new( conf, options[:tablename] )

scan = Scan.new
kfilters = FilterList.new(FilterList::Operator::MUST_PASS_ALL)
if options[:use_rowfilter]
then
  comp = RegexStringComparator.new(options[:delete_pattern])
  kfilters.addFilter(RowFilter.new(CompareFilter::CompareOp::EQUAL, comp))
end
kfilters.addFilter(FirstKeyOnlyFilter.new())
kfilters.addFilter(KeyOnlyFilter.new(true))
scan.setFilter(kfilters)
scan.setBatch(5000)
scan.setCacheBlocks(false)
scanner = table.getScanner( scan )
counter = 0
begin
  while row = scanner.next do
    key = Bytes.toString( row.getRow )
    if key =~ /#{options[:delete_pattern]}/
      puts " * Deleting \"#{key}\"" if options[:verbose]
      r = Delete.new(row.getRow)
      table.delete(r)
      counter += 1
      puts "#{counter} rows deleted" if counter % 100000 == 0
    elsif options[:suspects_antipattern] != '' and key !~ /#{options[:suspects_antipattern]}/
      puts " * Suspect key: \"#{key}\"" if (options[:verbose] or not options[:delete_suspects])
      if options[:delete_suspects]
        r = Delete.new(row.getRow)
        table.delete(r)
        counter += 1
        puts "#{counter} rows deleted" if counter % 100000 == 0
      end
    end
  end
ensure
    scanner.close
end
table.close
puts "Final count: #{counter} rows deleted"
