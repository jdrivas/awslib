package awslib

import(
  "fmt"
  "testing"
)

func TestECSConfiguration(t *testing.T) {
  clusterName := "test_cluster"
  configString, err := getECSConfigString(clusterName)
  if err != nil { t.Error("failed: %s", err) }
  fmt.Printf("Congiuration looks like: %s", configString)
}