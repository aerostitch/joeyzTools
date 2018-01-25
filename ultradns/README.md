# UltraDNS-related scripts

## `ultradns_import_to_terraform.py`

This scripts exports all Ultradns records of all your zones to a single
terraform config file and also generates the corresponding terraform.tfstate file.

### Prerequisistes (in a venv):
```
pip3 install ultra_rest_client
```

### Usage

The `-h` flag gives you the available options and their explanations:
```
$ python3 ultradns/ultradns_import_to_terraform.py -h
usage: ultradns_import_to_terraform.py [-h] [-a RES_API_URL] -u RES_API_USER
                                       -p RES_API_PASSWORD
                                       [-t TERRAFORM_VERSION]

Generating terraform file containing the current config you have in UltraDNS

optional arguments:
  -h, --help            show this help message and exit
  -a RES_API_URL, --rest-api-url RES_API_URL
                        Rest API endpoint to use to call the ultraDNS API.
                        Defaults to restapi.ultradns.com
  -u RES_API_USER, --rest-api-user RES_API_USER
                        User used to connect to the UltraDNS API
  -p RES_API_PASSWORD, --rest-api-password RES_API_PASSWORD
                        Password used to connect to the UltraDNS API
  -t TERRAFORM_VERSION, --terraform-version TERRAFORM_VERSION
                        Terraform version to specify in the resulting
                        terraform.state
```

Once the script has run, it will generate the config file (`ultradns.tf`) and
the tfstate (`terraform.tfstate`) inside your current working directory.

It also creates a `credentials.tf` file so you can gpg it.

Then you just have to make the modifications you want in your config and run a
`terraform plan` and `terraform apply`
