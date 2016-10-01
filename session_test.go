package awslib

import(
  // "fmt"
  // "os"
  "testing"
  // "strings"
  // "time"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/stretchr/testify/assert"  
)

const defaultProfile = "momentlabs-test"
func getSession() (sess *session.Session) {
  s, _ := GetSession(defaultProfile)
  return s
}

func TestSessionAndAccount(t *testing.T)  {
  skipOnShort(t)
  s, err := GetSession(defaultProfile)
  assert.NoError(t, err, "Failed to create a default session.")
  config := s.Config
  assert.NotNil(t, config.Region)
  assert.Equal(t, "us-east-1", *config.Region)

  aliases, err := GetAccountAliases(s.Config)
  assert.NoError(t, err, "Failed to get account aliases.")
  assert.Equal(t, "momentlabs", *aliases[0])

  ident, err := GetCurrentAccountIdentity(s)
  assert.NoError(t, err, "Failed to get account idenitty.")
  assert.Contains(t, *ident.Arn, "MainTest")
}

