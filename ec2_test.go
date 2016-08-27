package awslib

import(
  // "fmt"
  "testing"
  "github.com/stretchr/testify/assert"
)

func TestGetConfigFileString(t *testing.T) {
  _, err := getConfigFileString()
  if assert.NoError(t, err, "Can't get configuration file.") {
    // This is slightly tricky to get.
    // I'm going to juse go with running it, making sure
    // there are no errors on running it. 
    // If problems come up, the uncommenting the below may help.
    // fmt.Printf("ConfigFile:\n%s\n", s)
  }
}