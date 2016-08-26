package awslib

import (
  // "strings"
  "fmt"
  // "errors"
  // "time"  
  // "io"
  "os"
  "os/user"
  "path/filepath"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/credentials"
  "github.com/aws/aws-sdk-go/aws/defaults"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/iam"
  "github.com/Sirupsen/logrus"
)

// TODO: Verify that this  "does the right thing" if the creds file doesn't exist.
// Also, consider filling it out by looking at a 'config' file as well.
// profile is an AWS profile to use for creds, crefile is where creds
// are stored as profiles. You can specify a credFile of "" for
// the default of ~/.aws/credentials to be used.
// The thing to do here, is to create sessions and pass them around rather than configs.
// Sessions allows you to use the configuration file.
// http://docs.aws.amazon.com/sdk-for-go/api/aws/session/

// Look for credentials in a credential file provide by the credFile argument.
// If that string is "", look in ~/.aws/.credentials
// If there is no cred file then check the environment.

func GetConfig(profile string, credFile string) (*aws.Config) {
  config := defaults.Get().Config

  user, err  :=  user.Current()
  if err != nil {
    fmt.Printf("ecs-pilot: Could't get the current user from the OS: \n", err)
    os.Exit(1)
  }

  if credFile == "" { 
    credFile = filepath.Join(user.HomeDir, "/.aws/credentials")
  }
  logFields := logrus.Fields{"credential-file": credFile}

  _, err = os.Open(credFile)
  if err == nil {
    log.Debug(logFields,"Loading credentials.")
    creds := credentials.NewSharedCredentials(credFile, profile)  
    config.Credentials = creds
  } else {
    logFields["error"] = err
    log.Debug(logFields, "Can't load credentials from file.")
    log.Debug(nil, "Loading credentials from environment.")
    creds := credentials.NewEnvCredentials()
    config.Credentials = creds
  }

  // THIS SHOULD NEVER END UP IN PRODUCTION.
  // IT PRINTS OUT KEYS WHICH WOULD END UP IN LOGS.
  // credValue, err := config.Credentials.Get()
  // if err == nil {
  //   log.Debugf("Value of credential is: %#v", credValue)
  // } else {
  //   log.Debugf("Couldn't get the value of the credentials: %s", err)
  // }

  if *config.Region == "" {
    config.Region = aws.String("us-east-1")
  }
  return config
}

func GetDefaultConfig() (*aws.Config) {
  profile := os.Getenv("AWS_PROFILE")
  if profile == "" {
    profile = "default"
  }
  return GetConfig(profile, "")
}

func GetAccountAliases(config *aws.Config) (aliases []*string, err error) {

  iamSvc := iam.New(session.New(config))
  params := &iam.ListAccountAliasesInput{
    MaxItems: aws.Int64(100),
  }
  resp, err := iamSvc.ListAccountAliases(params)
  if err == nil {
    aliases = resp.AccountAliases
    if *resp.IsTruncated {
      log.Warn(nil, "More account aliases available, only got the first few.")
    }
  }
  return aliases, err
}

func AccountDetailsString(config *aws.Config) (details string, err error) {

  aliases, err := GetAccountAliases(config)
  if err == nil {
    if len(aliases) == 1 {
      details += fmt.Sprintf("Account: %s", *aliases[0])
    } else {
      details += fmt.Sprintf("Account: ")
      for i, alias := range aliases {
        details += fmt.Sprintf("%d. %s ", i+1, *alias)
      }
    }
  }
  details += fmt.Sprintf(" Region: %s", *config.Region)
  return details, err
}