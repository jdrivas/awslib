package awslib

import (
  "fmt"
  "testing"
  "strings"
  "time"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/stretchr/testify/assert"  
)


// Support functions
func skipOnShort(t *testing.T) {
  if testing.Short() { t.SkipNow() }
}


func testSession(t *testing.T) (sess *session.Session ) {
  testProfile := "mclib-test"
  s, err := session.NewSessionWithOptions(session.Options{
    Profile: testProfile,
    SharedConfigState: session.SharedConfigEnable,  
  })
  if assert.NoError(t, err) {
    sess = s
  }
  return sess
}

//
// Testing Helpers.
//
func TestShortDurationString(t *testing.T) {
  trueValues := []struct{ 
    d time.Duration
    expect string
  }{
    {d: 0 *time.Second, expect: "0m 0s"},
    {d: 1 *time.Second, expect: "0m 1s"},
    {d: 1 *time.Minute, expect: "1m 0s"},
    {d: 1 *time.Hour, expect: "1h 0m"},
    {d: 24 *time.Hour, expect: "1d 0h"},
  }

  falseValues := []struct{ 
    d time.Duration
    expect string
  }{
    {d: 0 *time.Second, expect: "0h 0m"},
    {d: 1 *time.Second, expect: "0h 0m"},

    {d: 1 *time.Minute, expect: "0m 60s"},
    {d: 1 *time.Minute, expect: "0h 1m"},

    {d: 1 *time.Hour, expect: "60m 0s"},
    {d: 1 *time.Hour, expect: "0d 1h"},

    {d: 24 *time.Hour, expect: "0d 24h"},
    {d: 24 *time.Hour, expect: "24h 0m"},
  }



  for _, tv := range trueValues {
    sds := ShortDurationString(tv.d)
    failure := fmt.Sprintf("For %d expected: %s, got: %s.", tv.d, tv.expect, sds)
    assert.True(t, strings.Contains(sds, tv.expect), failure)
  }

  for _, fv := range falseValues {
    sds := ShortDurationString(fv.d)
    failure := fmt.Sprintf("For %d expected: %s, got: %s.", fv.d, fv.expect, sds)
    assert.False(t, strings.Contains(sds, fv.expect), failure)
  }
}