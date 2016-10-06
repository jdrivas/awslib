package awslib

import (
  // "fmt"
  "testing"
  // "strings"
  // "time"
  "github.com/stretchr/testify/assert"  
)

func TestGetZoneString(t *testing.T) {
  testEqual:= []struct{
    input string
    expected string
  }{
    {input: "foo.bar.momentlabs.io", expected: "momentlabs.io",},
    {input: "foo.momentlabs.io", expected: "momentlabs.io",},
    {input: "foo.momentlabs.io.", expected: "momentlabs.io",},
  }

  for _, ts := range testEqual {
    res, ok := getZoneString(ts.input)
    assert.True(t, ok, "Couldn't get a zone for: %s", ts.input)
    assert.Equal(t,ts.expected, res)
  }

  testNotOk := []string{"moment", ".com",}
  for  _, ts := range testNotOk {
    res, ok := getZoneString(ts)
    assert.False(t, ok, "Returned %s for: %s", res, ts)
  }
}

const testIP = "127.0.0.1"
const testFQDN = "unit-test.test.momentlabs.io"
func TestDNSAttachDetach(t *testing.T) {
  skipOnShort(t)
  s := getSession()
  _, err := AttachIpToDNS(testIP, testFQDN, "Testing add a record with awslib.", 60, s)
  assert.NoError(t, err, "Failed to update DNS with a record entry.")

  _, err = DetachFromDNS(testIP, testFQDN, "Testing removing a record with awslib.", 60,  s)
  assert.NoError(t, err, "Failed to remove the DNS record entry.")
}