package awslib

import(
  "strings"
)

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