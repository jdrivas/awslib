package awslib

import(
  "testing"
  "github.com/stretchr/testify/assert"
)

func TestECSConfiguration(t *testing.T) {
  clusterName := "test_cluster"
  configString, err := getECSConfigString(clusterName)
  if assert.NoError(t, err){
    assert.Contains(t, configString, "ECS_CLUSTER=")
  }
}