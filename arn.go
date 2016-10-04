package awslib

import (
  "fmt"
  "strings"
  "github.com/aws/aws-sdk-go/aws/session"
)


// TODO: Fix this for more general use.
func ShortArnString(arn *string) (s string) {
  if arn == nil {
    return "<nil>"
  }
  splits := strings.Split(*arn, "/")
  shortArn := splits[0]
  if len(splits) >= 2 {
    shortArn = splits[1]
  }
  return shortArn
}

type ResourceType int
const (
  ContainerInstanceType ResourceType = iota
)

type resourceArnValues struct {
  typeString string
  servicePrefix string
}

func makeServicePrefix(service string) (string) {
  return fmt.Sprintf("arn:aws:%s:", service)
}
var arnResourceMap = map[ResourceType]resourceArnValues{
  ContainerInstanceType: {typeString: "container-instance", servicePrefix: makeServicePrefix("ecs")},
}

func makeLong(prefix, region, an, resource, shortArn string) (string) {
  return fmt.Sprintf("%s%s:%s:%s/%s", prefix, region, an, resource, shortArn)
}

// TODO: Probably should add simple test for isLong or isShort (isLikelyLong?)
func LongArnString(shortArn string, rt ResourceType, sess *session.Session ) (arn string, err error) {
  an, err := GetCurrentAccountNumber(sess)
  if err == nil {
    region := sess.Config.Region
    if region == nil || *region == "" { return arn, fmt.Errorf("LongArnString: failed to get a region from session.")}
    av := arnResourceMap[rt]
    arn = makeLong(av.servicePrefix, *region, an, av.typeString, shortArn)
  }

  return arn, err
}