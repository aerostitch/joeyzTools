package main

// This script either list all keys on an account
// or list the user corresponding to the given key

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

func main() {

	aws_key_filter := flag.String("filter-key", "", "Aws access key to look for. If not provided, will list all of them.")
	flag.Parse()

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	cli := iam.New(sess)
	userLimit := int64(1000) // 1000 is the max
	userParams := &iam.ListUsersInput{MaxItems: &userLimit}
	users, err := cli.ListUsers(userParams)
	if err != nil {
		panic(err)
	}

	//fmt.Println(len(users.Users), len(*aws_key_filter))
	for _, user := range users.Users {
		// Checks for the ley of the user
		keyParams := &iam.ListAccessKeysInput{UserName: user.UserName}
		keys, err := cli.ListAccessKeys(keyParams)
		if err != nil {
			panic(err)
		}
		for _, key := range keys.AccessKeyMetadata {
			if len(*aws_key_filter) > 0 {
				if *(key.AccessKeyId) == *aws_key_filter {
					fmt.Println(*(key.UserName), "-", *(key.AccessKeyId))
					return
				}
			} else {
				fmt.Println(*(key.UserName), "-", *(key.AccessKeyId))
			}
		}
	}
}
