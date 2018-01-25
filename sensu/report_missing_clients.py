#!/usr/bin/env python3
'''
List AWS instances not registered in Sensu
'''
# pip install requests boto3
import argparse
import requests
import boto3

def aws_tags_to_hash(tags):
    '''
    aws_tags_to_hash transform an ec2 tags array structure to a hash.

    >>> from pprint import pprint as pp
    >>> aws_tags_to_hash([])
    {}
    >>> aws_tags_to_hash(None)
    {}
    >>> pp(aws_tags_to_hash([
    ...                     {'Key': 'foo', 'Value': 'Foo value'},
    ...                     {'Key': 'bar', 'Value': 'Bar value'}]))
    {'bar': 'Bar value', 'foo': 'Foo value'}
    >>> aws_tags_to_hash([{'wrong': 'foo', 'Value': 'Foo value'}])
    {}
    '''
    _result = {}
    if tags is not None:
        for tag in tags:
            if 'Key' in tag and 'Value' in tag:
                _result[tag['Key']] = tag['Value']
    return _result

def get_sensu_clients(api_host, api_port, api_user, api_pwd):
    r = requests.get('http://{}:{}/clients'.format(api_host, api_port),
                     auth=(api_user, api_pwd))
    clients = r.json()
    
    cli = {}
    for client in clients:
        zone = 'unknown'
        id = client['name']
        if 'zone' in client:
            zone = client['zone']
        if 'instance_id' in client:
            id = client['instance_id']
    
        if zone not in cli:
            cli[zone] = {id: {'name': client['name']}}
        else:
            cli[zone][id] = {'name': client['name']}
    
        if 'aws_account_id' in client:
            cli[zone][client['instance_id']]['aws_account_id'] = client['aws_account_id'],
    return cli

def check_all_instances(region=None, sensu_clients=None, sensu_api_host=''):
    res_setup = {'service_name': 'ec2'}
    if region is not None:
        res_setup['region_name'] = region
    ec2 = boto3.resource(**res_setup)
    for instance in ec2.instances.filter(Filters=[{
            'Name': 'instance-state-name',
            'Values': ['pending', 'running', 'stopping', 'stopped']}]):
        instance_zone = instance.placement['AvailabilityZone']
        if instance_zone not in sensu_clients or instance.instance_id not in sensu_clients[instance_zone]:
            tags = aws_tags_to_hash(instance.tags)
            found_key = False
            for key in ('hostname', 'Name', 'cluster'):
                if key in tags:
                    identifier = "{}: {} ({})".format(key, tags[key], instance.instance_id)
                    found_key = True
                    break
            if not found_key:
                identifier = "Instance {}".format(instance.instance_id)

            print('{} - zone: {} - is not reporting to Sensu {}'.format(
                identifier, instance_zone, sensu_api_host))

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='List AWS instances not registered in Sensu')
    parser.add_argument('-s', '--sensu-api-host', metavar='SENSU_API_HOST', default='localhost',
            help='Hostname or IP of the Sensu API')
    parser.add_argument('-P', '--sensu-api-port', type=int, metavar='SENSU_API_PORT', default=4567,
            help='Port of the Sensu API')
    parser.add_argument('-u', '--sensu-api-user', metavar='SENSU_API_USER', default='admin',
            help='Login to use to authenticate to the Sensu API')
    parser.add_argument('-p', '--sensu-api-password', metavar='SENSU_API_PWD', default='',
            help='Password to use to connect to the Sensu api')
    parser.add_argument('-r', '--aws-region', metavar='AWS_REGION', default='us-east-1',
            help='region to scan for instances')
    args = parser.parse_args()
    sensu_cli = get_sensu_clients(
            args.sensu_api_host, args.sensu_api_port,
            args.sensu_api_user, args.sensu_api_password)
    check_all_instances(args.aws_region, sensu_cli, args.sensu_api_host)

    # for z in sensu_cli:
    #     for c in sensu_cli[z]:
    #         print("{} - {} - {}".format(z, c, sensu_cli[z][c]))
