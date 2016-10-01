package awslib

import(
  "fmt"
  "testing"
  "github.com/stretchr/testify/assert"
)


func makeLongArn(prefix, region, an, typeString, shortArn string) (string) {
  return prefix + region + ":" + an + ":" + typeString + "/" + shortArn
}

func TestGetLongArnString(t *testing.T) {
  skipOnShort(t)

  sess, err := GetSession("momentlabs-test")
  if err != nil { assert.FailNow(t, "Error attempting to get AWS session: %s", err) }
  region := sess.Config.Region
  if region == nil || *region == "" { assert.FailNow(t, "Failed to get region from configuraiton.") }
  an, err := GetCurrentAccountNumber(sess)
  if err != nil { assert.FailNow(t, "Error attempting  to get account number: %s", err )}


  testTrue := []struct{
    rtype ResourceType
    shortArn string
    expected string
  }{
    { rtype: ContainerInstanceType, shortArn: "arbtiraryshortarn123",
      expected: makeLongArn(arnResourceMap[ContainerInstanceType].servicePrefix, *region, an,
                  arnResourceMap[ContainerInstanceType].typeString, "arbtiraryshortarn123"),
    }, 
  }

  var s string
  for i, test := range testTrue {
    s, err = LongArnString(test.shortArn, test.rtype, sess)
    if assert.NoError(t, err, "Error on iteration %d: %s", i+1, err ) {
      assert.Equal(t, test.expected, s, "Failed on iteration: %d", i+1)
      fmt.Printf("Expected: %s\nReceived: %s\n", test.expected, s)
    }
  }
}