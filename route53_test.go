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