package awslib

import(
  "fmt"
  "strings"
  "time"
  "github.com/aws/aws-sdk-go/service/ecs"
)

// Sigh.
func JoinStringP(sp []*string, sep string) (v string) {
  s := make([]string, len(sp))
  for i, p := range sp {
    s[i] = *p
  }
  return strings.Join(s, sep)
}

func ShortDurationString(d time.Duration) (s string) {
  days := int(d.Hours()) / 24
  hours := int(d.Hours()) % 24
  minutes := int(d.Minutes()) % 60
  seconds := int(d.Seconds()) % 60

  // fmt.Printf("Days: %d, hours: %d, minutes: %d, seconds: %d\n", days, hours, minutes, seconds)

  if days == 0 &&  hours == 0 {
    return fmt.Sprintf("%dm %ds", minutes, seconds)
  }

  if days == 0 {
    return fmt.Sprintf("%dh %dm", hours, minutes)
  }

  // Days != 0.
  return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)

  return s
}

func CollectContainerNames(containers []*ecs.Container) (string) {
  s := ""
  for _, container := range containers {
    s += fmt.Sprintf("%s ", *container.Name)
  }
  return strings.Trim(s, ",")
}

func CollectBindings(task *ecs.Task) (string) {
  s := ""
  for _, c := range task.Containers {
    bdgs := c.NetworkBindings
    if len(bdgs) == 0 {continue}
    s += *c.Name + ": "
    for _, b := range bdgs {
      s += fmt.Sprintf("%d->%d, ", *b.ContainerPort, *b.HostPort)
    }
  }
  s = strings.Trim(s,", ")
  return s
}