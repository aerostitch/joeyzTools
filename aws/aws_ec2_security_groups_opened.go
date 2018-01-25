package main

// This script list the security groups with ports opened to the world for IPv4

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const cidr = "0.0.0.0/0"

func main() {

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := ec2.New(sess)

	params := &ec2.DescribeSecurityGroupsInput{
		DryRun: aws.Bool(false),
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("ip-permission.cidr"),
				Values: []*string{aws.String(cidr)},
			},
		},
	}

	resp, err := svc.DescribeSecurityGroups(params)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for _, group := range resp.SecurityGroups {
		fmt.Printf("Group ID: %s - Group Name: %s", *group.GroupId, *group.GroupName)
		if group.VpcId != nil {
			fmt.Printf(" - VPC ID: %s", *group.VpcId)
		}
		fmt.Printf("\n")
		for _, perm := range group.IpPermissions {
			fmt.Printf("\tProtocol: %s", *perm.IpProtocol)
			if perm.ToPort != nil {
				fmt.Printf(" - port: %d", *perm.ToPort)
			}
			if perm.IpRanges != nil {
				fmt.Printf(" - IP:")
				for _, ips := range perm.IpRanges {
					fmt.Printf(" %s", *ips.CidrIp)
				}
			}
			if perm.Ipv6Ranges != nil {
				fmt.Printf("\tIP v6:")
				for _, ips := range perm.Ipv6Ranges {
					fmt.Printf(" %s", *ips.CidrIpv6)
				}
			}
			fmt.Printf("\n")
		}
	}
}
