#!/usr/bin/env python3
'''
This scripts exports all Ultradns records of all your zones to a single terraform
config file and also generates the corresponding terraform.tfstate file.

Prerequisistes (in a venv):
```
pip3 install ultra_rest_client
```

Once the script has run, it will generate the config file (`ultradns.tf`) and
the tfstate (`terraform.tfstate`) inside your current working directory.
It also creates a `credentials.tf` file so you can gpg it.
Then you just have to make the modifications you want in your config and run a
`terraform plan` and `terraform apply`

'''

import argparse
import ultra_rest_client

parser = argparse.ArgumentParser(description='Generating terraform file containing the current config you have in UltraDNS')
parser.add_argument('-a', '--rest-api-url', metavar='RES_API_URL', default='restapi.ultradns.com',
                    help='Rest API endpoint to use to call the ultraDNS API. Defaults to restapi.ultradns.com')
parser.add_argument('-u', '--rest-api-user', metavar='RES_API_USER', required=True,
                    help='User used to connect to the UltraDNS API')
parser.add_argument('-p', '--rest-api-password', metavar='RES_API_PASSWORD', required=True,
                    help='Password used to connect to the UltraDNS API')
parser.add_argument('-t', '--terraform-version', metavar='TERRAFORM_VERSION', default='0.7.6',
                    help='Terraform version to specify in the resulting terraform.state')
args = parser.parse_args()


def process_zone(zone_name, offset=0, last_zone=False):
    '''
    Process an ultradns zone record by record.
    '''
    rrsets = ultraclient.get_rrsets(zone_name, limit=1000, offset=offset)
    if 'rrSets' in rrsets:
        subidx = 0
        for recset in rrsets['rrSets']:
            subidx += 1
            process_record(recset)
            if (not last_zone) or (subidx != len(rrsets['rrSets']) and len(rrsets['rrSets']) != 1000):
                tfstate.write(',')
    if len(rrsets['rrSets']) == 1000:
        offset += 1000
        process_zone(zone_name=zone_name, offset=offset, last_zone=last_zone)


def process_record(rrset):
    '''
    Process a record by adding it to the terraform config file and the
    corresponding tfstate file
    '''
    rrtype = rrset['rrtype'].split(' ')[0]
    record_name = '{}{}{}'.format(
        rrset['ownerName'].replace('.', '_').replace('*', '_'),
        zone_name.replace('.', '_').replace('*', '_'),
        rrtype
        )
    target_file.write('resource "ultradns_record" "{}" {{\n'.format(record_name))
    target_file.write('\tzone = "{}"\n'.format(zone_name))
    target_file.write('\tname = "{}"\n'.format(rrset['ownerName']))
    target_file.write('\trdata = ["{}"]\n'.format('", "'.join(str(i).replace('"', '\\"') for i in rrset['rdata'])))
    target_file.write('\ttype = "{}"\n'.format(rrtype))
    target_file.write('\tttl = {}\n'.format(rrset['ttl']))
    target_file.write('}\n')
    tfstate.write('"ultradns_record.{}": {{\n'.format(record_name))
    tfstate.write('"depends_on": [],\n')
    tfstate.write('"primary": {\n')
    tfstate.write('"id": "{}",\n'.format(record_name))
    tfstate.write('"attributes": {{\n "zone": "{}",\n'.format(zone_name))
    tfstate.write('"id": "{}",\n'.format(record_name))
    tfstate.write('"ttl": "{}",\n'.format(rrset['ttl']))
    tfstate.write('"type": "{}",\n'.format(rrtype))
    tfstate.write('"name": "{}"\n'.format(rrset['ownerName']))
    tfstate.write('}\n}\n}')


ultraclient = ultra_rest_client.RestApiClient(args.rest_api_user, args.rest_api_password, host=args.rest_api_url)
account_details = ultraclient.get_account_details()
account_name = account_details[u'accounts'][0][u'accountName']
all_zones = ultraclient.get_zones_of_account(account_name, offset=0, reverse=True)

credentials = open('credentials.tf', 'w')
credentials.write('variable "ultradns_username" {\n')
credentials.write('\tdefault = "{}"\n'.format(args.rest_api_user))
credentials.write('}\n')
credentials.write('variable "ultradns_password" {\n')
credentials.write('\tdefault = "{}"\n'.format(args.rest_api_password))
credentials.write('}\n')
credentials.close()

tfstate = open('terraform.tfstate', 'w')
tfstate.write('{\n\t"version": 1,\n')
tfstate.write('\t"terraform_version": "{}",\n'.format(args.terraform_version))
tfstate.write('\t"serial": 1,\n')
tfstate.write('\t"lineage": "587c86b1-8331-48da-b591-5d19c961a7af",\n')
tfstate.write('\t"modules": [\n{\n"path": [ "root" ],\n')
tfstate.write('\t"outputs": {},\n')
tfstate.write('\t"resources": {\n')
with open('ultradns.tf', 'w') as target_file:
    target_file.write('provider "ultradns" {\n')
    target_file.write('\tusername = "${var.ultradns_username}"\n')
    target_file.write('\tpassword = "${var.ultradns_password}"\n')
    target_file.write('\tbaseurl  = "https://{}/"\n'.format(args.rest_api_url))
    target_file.write('}\n')
    idx = 0
    for zone in all_zones['zones']:
        idx += 1
        zone_name = zone['properties']['name']
        is_last_zone = False
        if idx == len(all_zones['zones']):
            is_last_zone = True
        process_zone(zone_name=zone_name, last_zone=is_last_zone)

tfstate.write('\t},\n')
tfstate.write('\t"depends_on": []\n')
tfstate.write('\t}\n]\n}\n')
tfstate.close()
