package awslib

import (
  "testing"
)

func skipOnShort(t *testing.T) {
  if testing.Short() { t.SkipNow() }
}