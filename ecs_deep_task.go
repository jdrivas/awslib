package awslib

import(
  "fmt"
  "strings"
  "time"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/ecs"
  "github.com/aws/aws-sdk-go/service/ec2"
  "github.com/Sirupsen/logrus"
)
// [TaskArn]DeepTask. 
// A collections of deep tasks indexed by TaskArn.
type DeepTaskMap map[string]*DeepTask

func (dtm DeepTaskMap) DeepTasks() (dts []*DeepTask) {
  dts = make([]*DeepTask, 0, len(dtm))
  for _, dt := range dtm {
    dts = append(dts, dt)
  }
  return dts
}

func (dtm DeepTaskMap) TaskArns() (arns []string) {
  arns = make([]string, len(dtm))
  i := 0
  for a, _ := range dtm {
    arns [i] = a
    i++
  }
  return arns
}

// We often need quite a lot of information with a task.
// Deep task goes and gets all of it.
// If this becomes a performance bottleneck then we can consdier
// implementing lazy evaluation of these pointers.
type DeepTask struct {                // The json is as intended, the locationName allows us to use the AWS jsonutil.BuildJSON routines.
  Task *ecs.Task                      `json:"task" locationName:"task"`
  Failure *ecs.Failure                `json:"failure" locationName:"failure"`
  TaskDefinition *ecs.TaskDefinition  `json:"taskDefinition" locationName:"taskDefinition"`
  CInstance *ecs.ContainerInstance    `json:"containerInstance" locationName:"containerInstance"`
  CIFailure *ecs.Failure              `json:"containerInstanceFailure" locationName:"containerInstanceFailure"`
  EC2Instance *ec2.Instance           `json:"ec2Instance" locationName:"ec2Instance"`
}


// TODO: Early optimization is the root of all evil. In this case it was just plain dumb. 
// Need to reserse the work with GetDeepTaskList so that it does all of the work, then 
// we construct the task map out of the list. Sigh, because while we're at it, the bigger job
// of changing the names is what we should do: GetDeepTasks => GetDeepTaskMap; GetDeepTaskList => GetDeepTasks

// Returns the tasks associated with this cluster as a map [TaskArn]*DeepTask
func GetDeepTasks(clusterName string, sess *session.Session) (dtm DeepTaskMap, err error) {
  dtm = make(DeepTaskMap)
  ctMap, err := GetAllTaskDescriptions(clusterName, sess)
  if err != nil {return dtm, fmt.Errorf("GetDeepTasks: No tasks for cluster \"%s\": %s", clusterName, err)}
  // Quitely eat errors here.
  ciMap, ec2Map, err := GetContainerMaps(clusterName, sess)
  for taskArn, ct := range ctMap {
    dt := new(DeepTask)
    dt.Task = ct.Task
    dt.Failure = ct.Failure
    if ct.Task != nil {
      task := ct.Task
      dt.CInstance = ciMap[*task.ContainerInstanceArn].Instance
      dt.CIFailure = ciMap[*task.ContainerInstanceArn].Failure
      if dt.CInstance != nil {
        dt.EC2Instance = ec2Map[*dt.CInstance.Ec2InstanceId]
      }
      // Cache and/or lazy evaluate?
      td,  err  := GetTaskDefinition(*dt.Task.TaskDefinitionArn, sess)
      if err != nil {return dtm, fmt.Errorf("Failed to get the task definition for task %s: %s", dt.Task.TaskArn, err)}
      dt.TaskDefinition = td
    }
    dtm[taskArn] = dt
  }
  return dtm, err
}

func GetDeepTaskList(clusterName string, sess *session.Session) (dtl []*DeepTask, err error) {
  dtm, err := GetDeepTasks(clusterName, sess)
  if err == nil { dtl = dtm.DeepTasks()}
  return dtl, err
}


// this is expensive. It makes 4 calls to AWS to get information.
func GetDeepTask(clusterName, taskArn string, sess *session.Session) (dt *DeepTask, err error) {
  dto, err := GetTaskDescription(clusterName, taskArn, sess)  // ecs.DescribeTasksOutput
  if err != nil { return dt, fmt.Errorf("GetDeepTask: failed to get description for %s:%s: %s", clusterName, taskArn, err)}
  dt, err = makeDeepTaskWith(clusterName, taskArn, dto, sess)
  return dt, err
}


// TODO: There are more of these to do ...... 
func (dt DeepTask) StartedAtString() (string) {
  if dt.Task.StartedAt == nil { return "--" }
  return dt.Task.StartedAt.Local().Format(time.RFC1123)
}

func (dt DeepTask) UptimeString() (string) {
  if dt.Task.StartedAt == nil { return "--"}
  uptime, _ := dt.Uptime()
  return ShortDurationString(uptime)
}

func (dt DeepTask) Uptime() (ut time.Duration, err error) {
  start := dt.Task.StartedAt
  if start != nil {
    ut = time.Since(*start)
  } else {
    err = fmt.Errorf("Empty ecs.Task.StartedAt can't compute uptime.")
  }
  return ut, err
}

func (dtI DeepTask) StartedAtLess(dtJ DeepTask) (bool) {
  ti := dtI.Task.StartedAt
  tj := dtJ.Task.StartedAt
  switch {
    case ti == nil && tj == nil: 
    r := strings.Compare(fmt.Sprintf("%s", &dtI), fmt.Sprintf("%s", &dtJ))
    switch {
      case r <= 0: return true
      case r > 0: return false
    }
    case ti == nil: return true
    case tj == nil: return false
  }
  return ti.Before(*tj)
}

func (dtI DeepTask) UptimeLess(dtJ DeepTask) (bool) {
  uI, eI := dtI.Uptime()
  uJ, eJ := dtJ.Uptime()
  switch {
  case eI != nil && eJ != nil: { return false }
  case eI != nil: { return true }
  case eJ != nil: { return false }
  }
  return uI < uJ
}

func(dt DeepTask) LastStatus() (string) {
  s := "<unavailable>"
  if dt.Task.LastStatus != nil {
    s = *dt.Task.LastStatus
  }
  return s
}

func (dt DeepTask) TimeToStartString() (string) {
  if dt.Task.StartedAt == nil { return "--" }
  return ShortDurationString(dt.TimeToStart())
}

// TODO: decide if returning 0 or err is better.
func (dt DeepTask) TimeToStart() (time.Duration) {
  t := dt.Task 
  if t.StartedAt == nil || t.CreatedAt == nil {return 0 * time.Second}
  return t.StartedAt.Sub(*t.CreatedAt)
}

// Returns the address of the EC2Instance that the task is running on.
// This comes from a string pointer in the EC2Instance struct.
// There is a de-reference in here that could panic if the pointer is nil.
// This could happen if the task is not yet mapped to a ContainerInstance.
func (dt DeepTask) PublicIpAddress() (string) {
  return *dt.EC2Instance.PublicIpAddress
}

func (dt DeepTask) PrivateIpAddress() (string) {
  return *dt.EC2Instance.PrivateIpAddress
}

func (dt DeepTask) GetInstanceID() (*string) {
  return dt.EC2Instance.InstanceId
}

// Returns the host binding to a port.
func (dt DeepTask) PortHostBinding(containerName string, containerPort int64) (hostPort int64, ok bool) {
  bindings := dt.NetworkBindings(containerName)
  for _, binding := range bindings {
    if containerPort == *binding.ContainerPort {
      hostPort = *binding.HostPort
      ok = true
    }
  }
  return hostPort, ok
}

func (dt DeepTask) ContainerNamesString() (string) {
  return CollectContainerNames(dt.Task.Containers)
}

func (dt DeepTask) NetworkBindings(containerName string ) (bindings []*ecs.NetworkBinding) {
  for _, cntr := range dt.Task.Containers {
    if containerName == *cntr.Name {
      bindings = cntr.NetworkBindings
      break
    }
  }
  return bindings
}

// Returns the short verison of the ClusterARN from the task.
func (dt DeepTask) ClusterName() (cn string) {
  if dt.Task == nil {return "<no-cluster>"}
  return ShortArnString(dt.Task.ClusterArn)
}

// Retruns the names container or ok == false if it's not there
func (dt DeepTask) GetContainer(containerName string) (c *ecs.Container, ok bool)  {
  if dt.Task == nil { return c, false }
  for _, cntr := range dt.Task.Containers {
    if *cntr.Name == containerName {
      c = cntr
      ok = true
      break
    }
  }
  return c, ok
}

// TODO: Be careful, this impedence match between the aws-sdk and what
// we return here could prove costly ......
func (dt DeepTask) GetEnvironment(containerName string) (env map[string]string, ok bool) {

  env, ok  = dt.EnvironmentNoOverrides(containerName)
  if ok {
    to := dt.Task.Overrides
    if to == nil { return env, false }
    var kvps []*ecs.KeyValuePair
    cos := to.ContainerOverrides
    for _, co := range cos {
      if *co.Name == containerName {
        kvps = co.Environment
        ok = true
        break
      }
    }
    overrides := keyValuesToMap(kvps)

    // Merge overrides with Env. .. Add updates to environment.
    for key, value := range  overrides {
      env[key] = value
    }
  }
  return env, ok
}

// Environment for the container before any applied Overrides (e.g. as defined in the Task Definition)
func (dt DeepTask) EnvironmentNoOverrides(containerName string) (cenv map[string]string, ok bool) {
  cdef, ok := GetContainerDefinition(containerName, dt.TaskDefinition)
  if ok { 
    cenv = keyValuesToMap(cdef.Environment)
  }
  return cenv, ok 
}

func(dt DeepTask) EnvironmentFromNames(containers []string) (env map[string]string, ok bool) {
  for _, cn := range containers {
    e, k := dt.GetEnvironment(cn)
    if k {
      env = e
      ok = true
      break
    }
  }
  return env, ok
}

func keyValuesToMap(kvps []*ecs.KeyValuePair) (env map[string]string) {
  env = make(map[string]string, len(kvps))
  for _, kvp := range kvps {
    env[*kvp.Name] = *kvp.Value
  }
  return env
}

func makeDeepTaskWith(clusterName, taskArn string, dto *ecs.DescribeTasksOutput, sess *session.Session) (dt *DeepTask, err error) {

  // Get ContainerTasks indexed by taskArn. 
  // It's possible that more than one comes back so we have to deal with that.
  ctMap := makeCTMapFromDescribeTasksOutput(dto)

  // Let's only use the one based on the taskArn. Any other's we got back we'll ignore.
  // I hope this doesn't come to bite us (perhaps we'll never get extra's back.)
  if len(ctMap) > 1 {
    log.Debug(logrus.Fields{"numberOfTasks": len(ctMap),}, "We got more than one task with our request.")
  }

  // fmt.Printf("Looking for TaskArn: %s in:\n %#v\n", taskArn, ctMap)

  ct, ok := ctMap[taskArn]
  if !ok { return nil, fmt.Errorf("Failed to find the taskArn in the map for: %s.", taskArn)}

  ciMap, ec2Map, err := GetContainerMaps(clusterName, sess)

  // TODO: Refactor this stanza and it's cousin in GetDeepTasks (the DeepTaskMap one.)
  dt = new(DeepTask)
  dt.Task = ct.Task
  dt.Failure = ct.Failure
  if ct.Task != nil {
    task := ct.Task 
    dt.CInstance = ciMap[*task.ContainerInstanceArn].Instance
    dt.CIFailure = ciMap[*task.ContainerInstanceArn].Failure
    if dt.CInstance != nil {
      dt.EC2Instance = ec2Map[*dt.CInstance.Ec2InstanceId]
    }
    td, err := GetTaskDefinition(*task.TaskDefinitionArn, sess)
    if err != nil {
      return dt, fmt.Errorf("Failed to get task-definition for task %s: %s", taskArn, err)
    }
    dt.TaskDefinition = td
  }
  if ct.Task == nil && ct.Failure == nil {
    return dt, fmt.Errorf("Could not find task or failure for %s.", taskArn)
  }
  return dt, err
}


// DeepTask Sorting 
type deepTaskSort struct {
  dts []*DeepTask
  less func( dtI, dtJ *DeepTask) (bool)
}
func (s deepTaskSort) Len() int { return len(s.dts) }
func (s deepTaskSort) Swap(i, j int) { s.dts[i], s.dts[j] = s.dts[j], s.dts[i] }
func (s deepTaskSort) Less(i, j int) bool { return s.less(s.dts[i], s.dts[j]) }

func ByUptime(dtl []*DeepTask) (deepTaskSort) {
  return deepTaskSort{
    dts: dtl,
    less: func(dtI, dtJ *DeepTask) (bool) { return dtI.UptimeLess(*dtJ) },
  }
}

func ByStartedAt(dtl []*DeepTask) (deepTaskSort) {
  return deepTaskSort{
    dts: dtl,
    less: func(dtI, dtJ *DeepTask) (bool) { return dtI.StartedAtLess(*dtJ) },
  }
}
