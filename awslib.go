package awslib

import (
"github.com/jdrivas/sl"
"github.com/Sirupsen/logrus"
)

var(
  log = sl.New()
  awslibConfig libConfig
)

// Logs

func init() {
  defaultConfigureLogs()
  awslibConfig = NewConfig()
}

func SetLogLevel(l logrus.Level) {
  log.SetLevel(l)
}


func SetLogFormatter(f logrus.Formatter) {
  log.SetFormatter(f)
}

func defaultConfigureLogs() {
  formatter := new(sl.TextFormatter)
  formatter.FullTimestamp = true
  log.SetFormatter(formatter)
  log.SetLevel(logrus.InfoLevel)
}

//
// Sketchy configuration machineary. Waiting for config with files etc.
// Viper?
//
type libConfig map[string]string

const(
  InstCredFileKey = "instance-cred-file"
  InstCredProfileKey = "instance-cred-profile"
  InstDefaultRegionKey ="instance-region-key"
)

func NewConfig() (libConfig) {
  config := make(libConfig)
  defaults := [][2]string{
    {InstCredFileKey,"./.instance_credentials"},
    {InstCredProfileKey, "minecraft"},
    {InstDefaultRegionKey, "us-east-1"},
  }
  for _, d := range defaults {
    config[d[0]] = d[1]
  }
  return config
}
