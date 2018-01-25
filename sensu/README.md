# Sensu-related scripts

## `report_missing_clients.py`

List AWS instances not registered in Sensu
usage:
```
 report_missing_clients.py [-h] [-s SENSU_API_HOST] [-P SENSU_API_PORT]
                                 [-u SENSU_API_USER] [-p SENSU_API_PWD]
                                 [-r AWS_REGION]
```


```
optional arguments:
  -h, --help            show this help message and exit
  -s SENSU_API_HOST, --sensu-api-host SENSU_API_HOST
                        Hostname or IP of the Sensu API
  -P SENSU_API_PORT, --sensu-api-port SENSU_API_PORT
                        Port of the Sensu API
  -u SENSU_API_USER, --sensu-api-user SENSU_API_USER
                        Login to use to authenticate to the Sensu API
  -p SENSU_API_PWD, --sensu-api-password SENSU_API_PWD
                        Password to use to connect to the Sensu api
  -r AWS_REGION, --aws-region AWS_REGION
                        region to scan for instances
```
