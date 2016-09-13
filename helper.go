package awslib

import(
  "fmt"
  "time"
)


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