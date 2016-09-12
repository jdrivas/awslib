package awslib 

import (
  "fmt"
  "errors"
  "io"
  "sort"
  "time"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/private/protocol/json/jsonutil"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/ecs"
  "github.com/aws/aws-sdk-go/service/ec2"
  "github.com/Sirupsen/logrus"
)

//
// CLUSTERS
//

func CreateCluster(clusterName string, svc *ecs.ECS) (*ecs.Cluster, error) {
  params := &ecs.CreateClusterInput{
    ClusterName: aws.String(clusterName),
  }
  resp, err := svc.CreateCluster(params)
  var cluster *ecs.Cluster
  if err == nil {
    cluster = resp.Cluster
  }
  return cluster, err
}

func DeleteCluster(clusterName string, svc *ecs.ECS) (*ecs.Cluster, error) {
  params := &ecs.DeleteClusterInput{
    Cluster: aws.String(clusterName),
  }
  resp, err := svc.DeleteCluster(params)
  var cluster *ecs.Cluster
  if err == nil {
    cluster = resp.Cluster
  }
  return cluster, err
}

func GetClusters(svc *ecs.ECS) ([]*string, error) {

  params := &ecs.ListClustersInput {
    MaxResults: aws.Int64(100),
  } // TODO: this only will get the first 100 ....
  output, err := svc.ListClusters(params)
  clusters := output.ClusterArns
  return clusters, err
}

func DescribeCluster(clusterName string, svc *ecs.ECS) ([]*ecs.Cluster, error) {
  
  params := &ecs.DescribeClustersInput {
    Clusters: []*string{aws.String(clusterName),},
  }

  resp, err := svc.DescribeClusters(params)
  return resp.Clusters, err
}

// func GetAllClusterDescriptions(ecsSvc *ecs.ECS) ([]*ecs.Cluster, error) {
func GetAllClusterDescriptions(sess *session.Session) (Clusters, error) {
  ecsSvc := ecs.New(sess)
  clusterArns, err := GetClusters(ecsSvc)
  if err != nil {return make([]*ecs.Cluster, 0), err}

  params := &ecs.DescribeClustersInput {
    Clusters: clusterArns,
  }
  
  resp, err := ecsSvc.DescribeClusters(params)
  return resp.Clusters, err
}


type Clusters []*ecs.Cluster
type ClusterSortType int
const(
  ByActivity ClusterSortType = iota
  ByReverseActivity
)

func (cs Clusters) Sort(t ClusterSortType) {
  fmt.Printf("Sorting Clusters\n.")
  switch t  {
  case ByActivity: sort.Sort(clusterByActivity(cs))
  case ByReverseActivity: sort.Sort(rClusterByActivity(cs))
  }
}

// TODO: This is disgusting. There has to be a better way.
type clusterByActivity []*ecs.Cluster
func (cs clusterByActivity) Len() int { return len(cs) }
func (cs clusterByActivity) Swap(i,j int) { cs[i], cs[j] = cs[j], cs[i] }
func (cs clusterByActivity) Less (i, j int) bool {
  if cs[i].Status == nil && cs[j].Status == nil {return *cs[i].ClusterArn < *cs[j].ClusterArn}
  if (cs[i].Status == nil) { return true}
  if (cs[j].Status == nil ) { return false }

  if *cs[i].Status != *cs[j].Status { return *cs[i].Status < *cs[j].Status }

  if *cs[i].RunningTasksCount != *cs[j].RunningTasksCount { return *cs[i].RunningTasksCount < *cs[j].RunningTasksCount }
  return *cs[i].PendingTasksCount < *cs[j].PendingTasksCount
}

type rClusterByActivity []*ecs.Cluster
func (cs rClusterByActivity) Len() int { return len(cs) }
func (cs rClusterByActivity) Swap(i, j int) { cs[i], cs[j] = cs[j], cs[i] }
func (cs rClusterByActivity) Less (j, i int) bool {
  if cs[i].Status == nil && cs[j].Status == nil {return *cs[i].ClusterArn < *cs[j].ClusterArn}
  if (cs[i].Status == nil) { return true}
  if (cs[j].Status == nil ) { return false }
  if *cs[i].Status != *cs[j].Status { return *cs[i].Status < *cs[j].Status }

  if *cs[i].RunningTasksCount != *cs[j].RunningTasksCount { return *cs[i].RunningTasksCount < *cs[j].RunningTasksCount }
  return *cs[i].PendingTasksCount < *cs[j].PendingTasksCount
}


//
// CONTAINER INSTANCES
//

// returns a list of Containerinstance ARNS.
func GetContainerInstances(clusterName string, sess *session.Session)([]*string, error) {
  ecsSvc := ecs.New(sess)
  params := &ecs.ListContainerInstancesInput {
    Cluster: aws.String(clusterName),
    MaxResults: aws.Int64(100),
  }
  resp, err := ecsSvc.ListContainerInstances(params)
  if err != nil { return []*string{}, err }

  return resp.ContainerInstanceArns, nil
}


// Capturing the fact that whenever you actually get the
// description you also get potential failures.
type ContainerInstance struct {
  Instance  *ecs.ContainerInstance
  Failure *ecs.Failure
}

// Keyed on ConatinerInstanceArn or Ec2InstanceId
type ContainerInstanceMap map[string]*ContainerInstance

func GetAllContainerInstanceDescriptions(clusterName string, sess *session.Session) (ContainerInstanceMap, error) {

  instanceArns, err := GetContainerInstances(clusterName, sess)
  if err != nil { return make(ContainerInstanceMap), err }

  if len(instanceArns) <= 0 {
    return make(ContainerInstanceMap), nil
  }

  ecsSvc := ecs.New(sess)
  params := &ecs.DescribeContainerInstancesInput {
    ContainerInstances: instanceArns,
    Cluster: aws.String(clusterName),
  }
  resp, err := ecsSvc.DescribeContainerInstances(params)
  return makeCIMapFromDescribeContainerInstancesOutput(resp), err
}

func GetContainerInstanceDescription(clusterName string, containerArn string, sess *session.Session) (ContainerInstanceMap, error) {
  ecsSvc := ecs.New(sess)
  params := &ecs.DescribeContainerInstancesInput{
    ContainerInstances: []*string{aws.String(containerArn)},
    Cluster: aws.String(clusterName),
  }
  resp, err := ecsSvc.DescribeContainerInstances(params)
  return makeCIMapFromDescribeContainerInstancesOutput(resp), err
}

//Returns a map keyed on ContainerInstanceArns
func makeCIMapFromDescribeContainerInstancesOutput(dcio *ecs.DescribeContainerInstancesOutput) (ContainerInstanceMap) {

  ciMap := make(ContainerInstanceMap)
  // Seperate out the conatiners instances ...
  for _, instance := range dcio.ContainerInstances {
    ci := new(ContainerInstance)
    ci.Instance = instance
    ciMap[*instance.ContainerInstanceArn] =  ci
  }
  // ... and failures.
  for _, failure := range dcio.Failures { 
    // TODO: There is a bug here if we return more than one failure for a conatiner.
    // This should be a list of failures, or we should read the SDK code and determine
    // that only failure is ever returned. This pattern is throughout this library.
    ci := ciMap[*failure.Arn]
    if ci == nil {
      ci := new(ContainerInstance)
      ci.Failure = failure
      ciMap[*failure.Arn] = ci
    } else {
      ci.Failure = failure
    }
  }
  return ciMap
}

func (ciMap ContainerInstanceMap) GetEc2InstanceIds() ([]*string) {
  ids := []*string{}
  for _, ci := range ciMap {
    if ci.Instance != nil {
      ids = append(ids, ci.Instance.Ec2InstanceId)
    }
  }
  return ids
}

// Returns a map keyed on EC2InstanceIds (note: thre will be no failures.)
func (ciMap ContainerInstanceMap) GetEc2InstanceMap() (ContainerInstanceMap) {
  ec2Map := make(ContainerInstanceMap)
  for _, ci := range ciMap {
    if ci.Instance != nil {ec2Map[*ci.Instance.Ec2InstanceId] = ci}
  }
  return ec2Map
}

func TerminateContainerInstance(clusterName string, containerArn string, ecs_svc *ecs.ECS, ec2Svc *ec2.EC2) (resp *ec2.TerminateInstancesOutput, err error) {

  // Need to get the container instance description in order to get the ec2-instanceID.
  params := &ecs.DescribeContainerInstancesInput{
    ContainerInstances: []*string{aws.String(containerArn)},
    Cluster: aws.String(clusterName),
  }
  dci_resp, err := ecs_svc.DescribeContainerInstances(params)
  if err != nil {
    return nil, err
  }

  instanceId := getInstanceId(dci_resp.ContainerInstances, containerArn)
  if instanceId == nil {
    errMessage := fmt.Sprintf("TerminateContainerInstance: Can't find the Ec2 Instance ID for container arn: %s", containerArn)
    err = errors.New(errMessage)
    resp = nil
  } else {
   resp, err = TerminateInstance(instanceId, ec2Svc)
  }

  return resp, err
}

// TODP: This needs to match on long and short InstanceIDs.
// It looks like you can constructed an instnceARN like this:
// arn:aws:ecs:us-east-1:033441544097:container-instance/6cf583b2-b09d-42e9-af5e-c2502271e372
// arn:aws:ecs:<region>:<accoount-id>:container-instance/<short-arn>
// Should be obtainable from sess.
func getInstanceId(containerInstances []*ecs.ContainerInstance, containerArn string) (instanceId *string) {
  for _, instance := range containerInstances {
    if *instance.ContainerInstanceArn == containerArn {
      instanceId = instance.Ec2InstanceId
    }
  }
  return instanceId
}

func WaitUntilContainerInstanceActive(clusterName string, ec2InstanceId string, sess *session.Session) (*ecs.ContainerInstance, error) {
  for {
    resp, err := GetAllContainerInstanceDescriptions(clusterName, sess)
    if err != nil {
      return nil, fmt.Errorf("WaitUntilContainerInstanceActive: failed to get instance desecription on %s with %s : %s", clusterName, ec2InstanceId, err)
    }

    ec2iMap := resp.GetEc2InstanceMap()
    if inst := ec2iMap[ec2InstanceId]; inst != nil {
      if inst.Instance.Status == nil {continue}
      if *inst.Instance.Status == "ACTIVE" { return inst.Instance, nil }
    }
    time.Sleep(2 * time.Second)
  }

  // We should never get here.
}

func OnContainerInstanceActive(clusterName string, ec2InstanceId string, sess *session.Session, do func(*ecs.ContainerInstance, error)) {
  go func() {
    ci, err := WaitUntilContainerInstanceActive(clusterName, ec2InstanceId, sess)
    do(ci, err)
  }()
}

//
// TASKS
//


// We often need quite a lot of information with a task.
// Deep task goes and gets all of it.
type DeepTask struct {
  Task *ecs.Task
  Failure *ecs.Failure
  CInstance *ecs.ContainerInstance
  CIFailure *ecs.Failure
  EC2Instance *ec2.Instance
}


// TODO: There are more of these to do ...... 

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

func (dt DeepTask) NetworkBindings(containerName string ) (bindings []*ecs.NetworkBinding) {
  // cntrs := dt.T.Containers
  var container *ecs.Container
  for _, cntr := range dt.Task.Containers {
    if containerName == *cntr.Name {
      container = cntr
      break
    }
  }
  return container.NetworkBindings
}

// TODO: Be careful, this impedence match between the aws-sdk and what
// we return here could prove costly ......
func (dt DeepTask) GetEnvironment(containerName string) (env map[string]string, err error) {
  to := dt.Task.Overrides
  if to == nil { return env, fmt.Errorf("No environment attached to task.")}
  var kvps []*ecs.KeyValuePair
  cos := to.ContainerOverrides
  for _, co := range cos {
    if *co.Name == containerName {
      kvps = co.Environment
      break
    }
  }
  env = keyValuesToMap(kvps)
  return env, err
}


func GetContainerMaps(clusterName string, sess *session.Session) (ciMap ContainerInstanceMap, ec2Map map[string]*ec2.Instance, err error) {
  // This is ContainerInstance indexed by ContainerInstanceARN
  ciMap, err = GetAllContainerInstanceDescriptions(clusterName, sess)
  if err != nil {
    return ciMap, ec2Map, 
      fmt.Errorf("Couldn't get the ContainerInstance for the cluster %s: %s", clusterName, err)
  }

  ec2Map, err = DescribeEC2Instances(ciMap, sess)
  if err != nil {
    return ciMap, ec2Map, 
      fmt.Errorf("Couldn't get the EC2 Instances for the cluster %s: %s", clusterName, err)
  }
  return ciMap, ec2Map, err
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
  ct, ok := ctMap[taskArn] 
  if !ok { return nil, fmt.Errorf("Couldn't find the task for: %s.", taskArn)}

  ciMap, ec2Map, err := GetContainerMaps(clusterName, sess)

  // TODO: Refactor this stanza and it's cousing in GetDeepTasks (the DeepTaskMap one.)
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
  }
  if ct.Task == nil && ct.Failure == nil {
    return dt, fmt.Errorf("Could not find task or failure for %s.", taskArn)
  }
  return dt, err
}

func GetDeepTask(clusterName, taskArn string, sess *session.Session) (dt *DeepTask, err error) {
  dto, err := GetTaskDescription(clusterName, taskArn, sess)  // ecs.DescribeTasksOutput
  if err != nil { return dt, fmt.Errorf("GetDeepTask: failed to get description for %s:%s: %s", clusterName, taskArn, err)}
  dt, err = makeDeepTaskWith(clusterName, taskArn, dto, sess)
  return dt, err
}

// [TaskArn]DeepTask. 
// A collections of deep tasks indexed by TaskArn.
type DeepTaskMap map[string]*DeepTask

func GetDeepTasks(clusterName string, sess *session.Session) (dtm DeepTaskMap, err error) {
  ecsSvc := ecs.New(sess)
  dtm = make(DeepTaskMap)
  ctMap, err := GetAllTaskDescriptions(clusterName, ecsSvc)
  if err != nil {return dtm, fmt.Errorf("GetDeepTasks: No tasks for cluster \"%s\": %s", clusterName, err)}

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
    }
    dtm[taskArn] = dt
  }
  return dtm, err
}

// DeepTaskMask sorting interface.
type DeepTaskSortType int
const(
  ByUptime DeepTaskSortType = iota
  ByReverseUptime
)

func (dtm DeepTaskMap) DeepTasks(st DeepTaskSortType) (dts []*DeepTask) {
  dts = make([]*DeepTask, 0, len(dtm))
  for _, dt := range dtm {
    dts = append(dts, dt)
  }
  switch st {
    case ByUptime: By(uptime).Sort(dts)
    case ByReverseUptime: By(reverseUptime).Sort(dts)
  }
  return dts
}

// Definition of a deepTask sort less function
type By func(dt1, dt2 *DeepTask) bool

// Sort uses the less functiom from by, and the stSorter to actually do a sort.
func (by By) Sort(dts []*DeepTask) {
  sorter := &dtSorter{
    dts: dts,
    by: by,
  }
  sort.Sort(sorter)
}
// dtSorter, this holds Len() and Swap() and keeps 
// a variable for a pluggable Less()
type dtSorter struct {
  dts []*DeepTask
  by func(dt1, dt2 *DeepTask) bool
}

// For sort ..
func (s *dtSorter) Len() int { return len(s.dts) }
func (s *dtSorter) Swap(i,j int) { s.dts[i], s.dts[j] = s.dts[j], s.dts[i] }
func (s *dtSorter) Less(i,j int) bool { return s.by(s.dts[i], s.dts[j]) }

var uptime = func(dt1, dt2 *DeepTask) bool {
  ut1, _ := dt1.Uptime()
  ut2, _ := dt2.Uptime()
  return ut1 < ut2
}

var reverseUptime = func(dt1, dt2 *DeepTask) bool {
  ut1, _ := dt1.Uptime()
  ut2, _ := dt2.Uptime()
  return ut2 < ut1
}

func ListTasksForCluster(clusterName string, ecs_svc *ecs.ECS) ([]*string, error) {

  params := &ecs.ListTasksInput{
    Cluster: aws.String(clusterName),
    MaxResults: aws.Int64(100),
  }
  resp, err := ecs_svc.ListTasks(params)
  return resp.TaskArns, err
}


type ContainerTask struct {
  Task *ecs.Task
  Failure *ecs.Failure
}

func (ct *ContainerTask) UptimeString() (string) {
  return ShortDurationString(ct.Uptime())
}

// This is since created as opposed to started.
// TODO: Consider StartedAt vs CreatedAt
func (ct *ContainerTask) Uptime() (time.Duration) {
  return time.Since(*ct.Task.CreatedAt)
}

func (ct *ContainerTask) TimeToStartString() (string) {
  return ShortDurationString(ct.TimeToStart())
}

func (ct *ContainerTask) TimeToStart() (time.Duration) {
  if ct.Task.StartedAt == nil || ct.Task.CreatedAt == nil {
    return 0 * time.Second
  }
  return ct.Task.StartedAt.Sub(*ct.Task.CreatedAt)
}

// indexed on taskARN
type ContainerTaskMap map[string]*ContainerTask


func GetAllTaskDescriptions(clusterName string, ecs_svc *ecs.ECS) (ContainerTaskMap, error) {
 
 taskArns, err := ListTasksForCluster(clusterName, ecs_svc)
 if err != nil { return make(ContainerTaskMap), err}

 // Describe task will fail with no arns.
 if len(taskArns) <= 0 {
  return make(ContainerTaskMap), nil
 }

  params := &ecs.DescribeTasksInput {
    Cluster: aws.String(clusterName),
    Tasks: taskArns,
  }

  resp, err := ecs_svc.DescribeTasks(params)
  return makeCTMapFromDescribeTasksOutput(resp), err
}

func GetTaskDescription(clusterName string, taskArn string, sess *session.Session) (*ecs.DescribeTasksOutput, error) {
  ecsSvc := ecs.New(sess)
  params := &ecs.DescribeTasksInput {
    Cluster: aws.String(clusterName),
    Tasks: []*string{aws.String(taskArn)},
  }
  resp, err := ecsSvc.DescribeTasks(params)
  return resp, err
}

// This just bunches up tasks by taskarn and failures by failure rn.
func makeCTMapFromDescribeTasksOutput(dto *ecs.DescribeTasksOutput) (ContainerTaskMap) {
  ctMap := make(ContainerTaskMap)
  for _, task := range dto.Tasks {
    ct := new(ContainerTask)
    ct.Task = task
    ctMap[*task.TaskArn] =  ct
  }
  for _, failure := range dto.Failures {
    ct := ctMap[*failure.Arn]
    if ct == nil {
      ct := new(ContainerTask)
      ct.Failure = failure
      ctMap[*failure.Arn] = ct
    } else {
      ct.Failure = failure
    }
  }
  return ctMap
}

// A Map of maps. 
// ContainerName -> [Key]Value
type ContainerEnvironmentMap map[string]map[string]string

func RunTaskWithEnv(clusterName string, taskDefArn string, envMap ContainerEnvironmentMap, ecsSvc *ecs.ECS) (*ecs.RunTaskOutput, error) {
  to := envMap.ToTaskOverride()
  params := &ecs.RunTaskInput{
    TaskDefinition: aws.String(taskDefArn),
    Cluster: aws.String(clusterName),
    Count: aws.Int64(1),
    Overrides: &to,
  }
  resp, err := ecsSvc.RunTask(params)
  if err != nil {err = fmt.Errorf("RunTaskWithEnv %s %s:  %s", clusterName, taskDefArn, err)}

  return resp, err
}

// ConatinerEnvironmentMap is environments keyed on containers nams.
// Environment is [key]:value (all strings).
func (envMap ContainerEnvironmentMap)ToTaskOverride() (to ecs.TaskOverride) {
  containerOverrides := []*ecs.ContainerOverride{}
  for containerName, env := range envMap {
    containerOverrides = append(containerOverrides, envToContainerOverride(containerName, env))
  }
  to.ContainerOverrides = containerOverrides
  return to
}

func envToContainerOverride(containerName string, env map[string]string) (co *ecs.ContainerOverride) {
  keyValues := envToKeyValues(env)
  co = &ecs.ContainerOverride{
    Environment: keyValues,
    Name: aws.String(containerName),
  }
  return co
}

func envToKeyValues(env map[string]string) (keyValues []*ecs.KeyValuePair) {
  for key, value := range env {
    keyValues = append(keyValues, &ecs.KeyValuePair{Name: aws.String(key), Value: aws.String(value)})
  }
  return keyValues
}

func keyValuesToMap(kvps []*ecs.KeyValuePair) (env map[string]string) {
  env = make(map[string]string, len(kvps))
  for _, kvp := range kvps {
    env[*kvp.Name] = *kvp.Value
  }
  return env
}


func RunTask(clusterName string, taskDef string, ecsSvc *ecs.ECS) (*ecs.RunTaskOutput, error) {
  env := make(ContainerEnvironmentMap)
  resp, err := RunTaskWithEnv(clusterName, taskDef, env, ecsSvc)
  return resp, err
}

func OnTaskRunning(clusterName, taskDefArn string, ecsSvc *ecs.ECS, do func(*ecs.DescribeTasksOutput, error)) {
    go func() {
      task_params := &ecs.DescribeTasksInput{
        Cluster: aws.String(clusterName),
        Tasks: []*string{aws.String(taskDefArn)},
      }
      err := ecsSvc.WaitUntilTasksRunning(task_params)
      td, newErr := ecsSvc.DescribeTasks(task_params)
      if err == nil { err = newErr }
      do(td, err)
    }()
}

func StopTask(clusterName string, taskArn string, sess *session.Session) (*ecs.StopTaskOutput, error)  {
  ecsSvc := ecs.New(sess)
  params := &ecs.StopTaskInput{
    Task: aws.String(taskArn),
    Cluster: aws.String(clusterName),
  }
  resp, err := ecsSvc.StopTask(params)
return resp, err
}

func OnTaskStopped(clusterName, taskArn string, sess *session.Session, do func(dto *ecs.DescribeTasksOutput, err error)) {
  go func() {
    ecsSvc := ecs.New(sess)
    waitParams := &ecs.DescribeTasksInput{
      Cluster: aws.String(clusterName),
      Tasks: []*string{aws.String(taskArn)},
    }
    err := ecsSvc.WaitUntilTasksStopped(waitParams)
    var dto *ecs.DescribeTasksOutput
    if err == nil {
      dto, err = GetTaskDescription(clusterName, taskArn, sess)
    }
    do(dto, err)
  }()
}


//
// Task Definitions
//

func ListTaskDefinitions(ecs_svc *ecs.ECS) ([]*string, error) {
  params := &ecs.ListTaskDefinitionsInput{
    MaxResults: aws.Int64(100),
  }
  resp, err := ecs_svc.ListTaskDefinitions(params)
  return resp.TaskDefinitionArns, err
}

func GetTaskDefinition(taskDefinitionArn string, ecs_svc *ecs.ECS) (*ecs.TaskDefinition, error) {
  params := &ecs.DescribeTaskDefinitionInput {
    TaskDefinition: aws.String(taskDefinitionArn),
  }
  resp, err := ecs_svc.DescribeTaskDefinition(params)
  return resp.TaskDefinition, err
}

// TODO: This relies on an unsupported JSON unmarshalling interface in the aws go-sdk.
// This could stop working.
func RegisterTaskDefinitionWithJSON(json io.Reader, ecs_svc *ecs.ECS) (*ecs.RegisterTaskDefinitionOutput, error) {
  var tdi ecs.RegisterTaskDefinitionInput
  err := jsonutil.UnmarshalJSON(&tdi, json)
  if err != nil { return nil, err}
  log.Debug(nil, "RegisterTaskDefinition: Decoded JSON stream.")

  resp, err := ecs_svc.RegisterTaskDefinition(&tdi)
  if err == nil {
    log.Debug(nil, "RegisterTaskDefinition: Registered Task.")
  }
  return resp, err
}

func DefaultTaskDefinition() (ecs.RegisterTaskDefinitionInput) {
    var tdi = ecs.RegisterTaskDefinitionInput{
    Family: aws.String("Family"),
    // TaskRoleArn: This appears not to be in the golang definition.
    ContainerDefinitions: []*ecs.ContainerDefinition{
      &ecs.ContainerDefinition{

        // REQUIRED Basic paramaters
        //
        Name: aws.String("Task Definition Name"),

        Image: aws.String("IMAGE REFERENCE"),
        // Maximum memory in MB (recomended 300-500 if unsure.)
        // Conatiner is killed if you try to exceed this amount of memory.
        Memory: aws.Int64(500),        // DOCKER CMD

        // Other Basic Components
        //
        Command: []*string{aws.String("CMD"),},
        // DOCKER Entrypoint.
        EntryPoint: []*string{
          aws.String("ENTRYPOINT"),
        },
        DockerLabels: nil,

        // Environment
        //
        // The number of CPUY units to reserve for this container, there are 1024 for each EC2 core.
        Cpu: aws.Int64(0),
        // If marked true, then the failure of this coantiner will stop the task.
        Essential: aws.Bool(true),
        // Working directory to run binaries from.
        WorkingDirectory: nil,
        // Environment Variables
        Environment: nil,

        // Networking
        //
        // When true, this means networking is disabled within the container.  (defaulat: false).
        DisableNetworking: aws.Bool(false),
        PortMappings: []*ecs.PortMapping{
          {
            ContainerPort: aws.Int64(25565),
            HostPort: aws.Int64(25565),
            Protocol: aws.String("tcp"),
          },
        },
        // Hostname to use for your container.
        Hostname: nil,
        // DNS Servers presented to the container.
        DnsServers: nil,
        // DNS Search domains presented to the container.
        DnsSearchDomains: nil,
        // Enties to append to /etc/hosts.
        ExtraHosts: nil,

        // Storage
        //
        // If true then the container is given only readonly access to the root filesystem.
        ReadonlyRootFilesystem: aws.Bool(false),
        // this is like the --volumes option in the docker run command.
        MountPoints: nil,
        VolumesFrom: nil,

        // Logs
        LogConfiguration: nil,

        // Security
        //
        // Elevated privileges when container is run - like root.
        Privileged: aws.Bool(false),
        // run commands as this user.
        User: nil,
        // Labels for SELinux and AppArmour 
        DockerSecurityOptions: nil,

        // Resource Limits
        //
        // A list of ulimits to set in the container.
        // (eg. CORE, CPU, FSIZE, LOCKS, MLOCK, MSGQUEUE, NICE, NFILE, NPROC, RSS, RTPRIO, RTTIME, SIGPENDING, STACK)
        Ulimits: nil,
      },
    },
    Volumes: []*ecs.Volume{},
  }
  return tdi
}

func CompleteEmptyTaskDefinition() (ecs.RegisterTaskDefinitionInput) {
  var tdi = ecs.RegisterTaskDefinitionInput{
    Family: aws.String(""),
    // TaskRoleArn: This appears not to be in the golang definition.
    ContainerDefinitions: []*ecs.ContainerDefinition{
      &ecs.ContainerDefinition{

        // REQUIRED Basic paramaters
        //
        Name: aws.String(""),

        Image: aws.String(""),
        // Maximum memory in MB (recomended 300-500 if unsure.)
        // Conatiner is killed if you try to exceed this amount of memory.
        Memory: aws.Int64(0),        // DOCKER CMD

        // Other Basic Components
        //
        Command: []*string{aws.String(""),},
        // DOCKER Entrypoint.
        EntryPoint: []*string{
          aws.String(""),
        },
        DockerLabels: map[string]*string {
          "Key": aws.String("Value"),
        },

        // Environment
        //
        // The number of CPUY units to reserve for this container, there are 1024 for each EC2 core.
        Cpu: aws.Int64(0),
        // If marked true, then the failure of this coantiner will stop the task.
        Essential: aws.Bool(true),
        // Working directory to run binaries from.
        WorkingDirectory: aws.String(""),
        // Environment Variables
        Environment: []*ecs.KeyValuePair{
          {
            Name: aws.String(""),
            Value: aws.String(""),
          },
        },

        // Networking
        //
        // When true, this means networking is disabled within the container.  (defaulat: false).
        DisableNetworking: aws.Bool(false),
        PortMappings: []*ecs.PortMapping{
          {
            ContainerPort: aws.Int64(1),
            HostPort: aws.Int64(1),
            Protocol: aws.String("tcp"),
          },
        },
        // Hostname to use for your container.
        Hostname: aws.String(""),
        // DNS Servers presented to the container.
        DnsServers: []*string{
          aws.String(""),
        },
        // DNS Search domains presented to the container.
        DnsSearchDomains: []*string{
          aws.String(""),
        },
        // Enties to append to /etc/hosts.
        ExtraHosts: []*ecs.HostEntry{
          {
            Hostname: aws.String(""),
            IpAddress: aws.String(""),
          },
        },

        // Storage
        //
        // If true then the container is given only readonly access to the root filesystem.
        ReadonlyRootFilesystem: aws.Bool(false),
        // this is like the --volumes option in the docker run command.
        MountPoints: []*ecs.MountPoint{
          {
            ContainerPath: aws.String(""),
            ReadOnly: aws.Bool(false),
            SourceVolume: aws.String(""),
          },
        },
        VolumesFrom: []*ecs.VolumeFrom{
          {
            ReadOnly: aws.Bool(false),
            SourceContainer: aws.String(""),
          },
        },

        // Logs
        LogConfiguration: nil,

        // Security
        //
        // Elevated privileges when container is run - like root.
        Privileged: aws.Bool(false),
        // run commands as this user.
        User: aws.String(""),
        // Labels for SELinux and AppArmour 
        DockerSecurityOptions: []*string{
          aws.String(""),
        },
        // Resource Limits
        //
        // A list of ulimits to set in the container.
        // (eg. CORE, CPU, FSIZE, LOCKS, MLOCK, MSGQUEUE, NICE, NFILE, NPROC, RSS, RTPRIO, RTTIME, SIGPENDING, STACK)
        Ulimits: []*ecs.Ulimit{
          {
            Name: aws.String(""),
            HardLimit: aws.Int64(1),
            SoftLimit: aws.Int64(1),
          },
        },
      },
    },
    Volumes: []*ecs.Volume{
      {
        Host: &ecs.HostVolumeProperties{
          SourcePath: aws.String(""),
        },
        Name: aws.String(""),
      },
    },
  }
  return tdi
}
// func WaitForContainerInstanceStateChange(delaySeconds, periodSeconds int, currentState string, 
//   clusterName string, conatinerIntstanceArn string, ecs_svc *ecs.ECS, cb func(string, error)) {
//   go func() {
//     time.Sleep(time.Second * time.Duration(delaySeconds))
//     var e error
//     var status string
//     for ci, e := s.GetContainerInstanceDescription(); e == nil;  {
//       if *sd.StreamStatus != currentState {
//         status = *sd.StreamStatus
//         break;
//       }
//       time.Sleep(time.Second * time.Duration(periodSeconds))
//       sd, e = s.GetAWSDescription()
//     }
//     cb(status, e)
//   }()
// }


//
// Containers
//

const (
  ContainerStateRunning = "RUNNING"
  ContainerStatePending = "PENDING"
)

// check for !running and !pending as opposed to just STOPPED.
func ContainerStatusOk(c *ecs.Container) bool {
  return *c.LastStatus == "PENDING" || *c.LastStatus == "RUNNING"
}

