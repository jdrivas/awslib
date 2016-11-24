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


// To add a new type, add the const below and then the definition in the arnResourceMap
type ResourceType int
const (
  ContainerInstanceType ResourceType = iota
  TaskDefinitionType
)

// The general ARN form is  "arn:aws:<service>:<region>:<account-number>:<resource-type>/<shortArn>"
// Example ARN: arn:aws:ecs:us-east-1:033441544097:task-definition/craft-logstash:4
// In this exapmle the typeString is the string "task-definiiton"
// the servicePrefix is constructed from the <service> and the session.Session passed, in this case "ecs'"
// Of course this will be very brittle and require a lot of following along if AWS makes changes.
// TODO: make this easier to fill out ideally: var arnResource [][]{ {TaskDefinitionType, "task-definition", "ecs"} }
// typedef resourceEntry struct {ResourceType, Type: string, Service: string}
// var arnEntryList{} []resourceEntry{
//  {ResourceType: TaskDefinitiontype, Type: "task-definition", Service: "ecs",},
//}
// Then fill out the arnResourceMap at init time.
var arnResourceMap = map[ResourceType]resourceArnValues{
  ContainerInstanceType: { typeString: "container-instance", servicePrefix: makeServicePrefix("ecs") },
  TaskDefinitionType: { typeString: "task-definition", servicePrefix: makeServicePrefix("ecs") },
}

type resourceArnValues struct {
  typeString string
  servicePrefix string
}

func makeServicePrefix(service string) (string) {
  return fmt.Sprintf("arn:aws:%s:", service)
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