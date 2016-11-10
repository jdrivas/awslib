package awslib 

import (
  "fmt"
  "errors"
  // "io"
  "sort"
  "strconv"
  "time"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/ecs"
  "github.com/aws/aws-sdk-go/service/ec2"
  // "github.com/Sirupsen/logrus"
)

//
// CLUSTERS
//

func CreateCluster(clusterName string, sess *session.Session) (*ecs.Cluster, error) {
  ecsSvc := ecs.New(sess)
  params := &ecs.CreateClusterInput{
    ClusterName: aws.String(clusterName),
  }
  resp, err := ecsSvc.CreateCluster(params)
  var cluster *ecs.Cluster
  if err == nil {
    cluster = resp.Cluster
  }
  return cluster, err
}

func DeleteCluster(clusterName string, sess *session.Session) (*ecs.Cluster, error) {
  ecsSvc := ecs.New(sess)
  params := &ecs.DeleteClusterInput{
    Cluster: aws.String(clusterName),
  }
  resp, err := ecsSvc.DeleteCluster(params)
  var cluster *ecs.Cluster
  if err == nil {
    cluster = resp.Cluster
  }
  return cluster, err
}

func GetClusters(sess *session.Session) ([]*string, error) {
  ecsSvc := ecs.New(sess)
  arns := make([]*string, 0)
  err := ecsSvc.ListClustersPages(&ecs.ListClustersInput{}, 
    func(page *ecs.ListClustersOutput, lastPage bool) bool {
      arns = append(arns, page.ClusterArns... )
      return true
  })
  return arns, err
}

func DescribeCluster(clusterName string, sess *session.Session) ([]*ecs.Cluster, error) {

  ecsSvc := ecs.New(sess)  
  params := &ecs.DescribeClustersInput {
    Clusters: []*string{aws.String(clusterName),},
  }

  resp, err := ecsSvc.DescribeClusters(params)
  return resp.Clusters, err
}

// func GetAllClusterDescriptions(ecsSvc *ecs.ECS) ([]*ecs.Cluster, error) {
func GetAllClusterDescriptions(sess *session.Session) (Clusters, error) {
  clusterArns, err := GetClusters(sess)
  if err != nil {return make([]*ecs.Cluster, 0), err}

  ecsSvc := ecs.New(sess)
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
  // fmt.Printf("Sorting Clusters.\n)
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

func (ciMap ContainerInstanceMap) InstanceCount() (int) {
  count := 0
  for _, ci := range ciMap {
    if ci.Instance != nil {count++}
  }
  return count
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

// Returns a map keyed on EC2InstanceIds
func (ciMap ContainerInstanceMap) GetEc2InstanceMap() (ContainerInstanceMap) {
  ec2Map := make(ContainerInstanceMap)
  for _, ci := range ciMap {
    if ci.Instance != nil {ec2Map[*ci.Instance.Ec2InstanceId] = ci}
  }
  return ec2Map
}

// Collect up the resources in the containers and return the totals.
func (ciMap ContainerInstanceMap) Totals() (r InstanceResource) {
  rg := make([]*ecs.Resource, 0)
  rm := make([]*ecs.Resource, 0)
  pt := int64(0)
  rt := int64(0)
  for _, ci := range ciMap {
    if ci.Instance != nil {
      i := ci.Instance
      rg = append(rg, i.RegisteredResources...)
      rm = append(rm, i.RemainingResources...)
      pt += *i.PendingTasksCount
      rt += *i.RunningTasksCount
    }
  }
  r = InstanceResource{
    Registered: collectResources(rg),
    Remaining: collectResources(rm),
    PendingTasks: pt,
    RunningTasks: rt,
  }
  return r
}

// Keyed on resource type.
type ResourceMap map[string]*ecs.Resource

// Might be better to consider hashes (on name) for the resource collecitons
type InstanceResource struct {
  Registered ResourceMap
  Remaining ResourceMap
  PendingTasks int64
  RunningTasks int64
}

func collectResources(rs []*ecs.Resource) (ResourceMap) {
  // cs := make([]*ecs.Resource, 0)

  // csMap := make(map[string]*ecs.Resource,0)
  csMap := make(ResourceMap, 0)
  for _, r := range rs {
    // Check to see if we already have the resource ...
    newr, ok := csMap[*r.Name]
    if !ok { // ... no, create a new one.
      newr = new(ecs.Resource)
      csMap[*r.Name] = newr
      newr.Name = r.Name
      newr.Type = r.Type
      switch *r.Type {
      case "INTEGER":
        newr.IntegerValue = new(int64)
      case "LONG": 
        newr.LongValue = new(int64)
      case "DOUBLE": 
        newr.DoubleValue = new(float64)
      case "STRINGSET": 
        newr.StringSetValue = make([]*string, 0, len(r.StringSetValue))
      }
    }

    // ... once we have the resource, add the new value to the old.
    switch *r.Type {
    case "INTEGER":
      *newr.IntegerValue += *r.IntegerValue
    case "LONG": 
      *newr.LongValue += *r.LongValue
    case "DOUBLE": 
      *newr.DoubleValue +=  *r.DoubleValue
    case "STRINGSET":
      newr.StringSetValue = append(newr.StringSetValue, r.StringSetValue...)
    }
  }

  // for _, r := range csMap {
  //   cs = append(cs, r)
  // }
  return csMap
}

// Returns the string representation of the value.
func (rm ResourceMap) StringFor(resourceName string) (v string) {
  v = "--"
  if r, ok := rm[resourceName]; ok {
    v = getValueString(r)
  }
  return v
}

// TODO: Should add a gather flag for totaling values in StringSet.
// e,g. ["1", "2", "3", "2", "1", "4"] => "1:2, 2:2, 3:1, 4:1" or something.
func getValueString(r *ecs.Resource) (v string) {
    switch *r.Type {
    case "INTEGER", "LONG": 
      v = strconv.FormatInt(*r.IntegerValue,10)
    case "DOUBLE": 
      v = strconv.FormatFloat(*r.DoubleValue, 'g', 2, 64)
    case "STRINGSET":
      v = JoinStringP(r.StringSetValue, ",")
  }
  return v
}



// Returns both the CotnainerInstanceMap (cis index by ciArn) and the ec2version ec2Is on ec2ID (not arn)
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


func ListTasks(clusterName string, sess *session.Session) ([]*string, error) {

  ecsSvc := ecs.New(sess)
  params := &ecs.ListTasksInput{
    Cluster: aws.String(clusterName),
    MaxResults: aws.Int64(100),
  }
  resp, err := ecsSvc.ListTasks(params)
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


func GetAllTaskDescriptions(clusterName string, sess *session.Session) (ContainerTaskMap, error) {
 
 taskArns, err := ListTasks(clusterName, sess)
 if err != nil { return make(ContainerTaskMap), err}

 // Describe task will fail with no arns.
 if len(taskArns) <= 0 {
  return make(ContainerTaskMap), nil
 }


 ecsSvc := ecs.New(sess)
  params := &ecs.DescribeTasksInput {
    Cluster: aws.String(clusterName),
    Tasks: taskArns,
  }
  resp, err := ecsSvc.DescribeTasks(params)
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

func RunTaskWithEnv(clusterName string, taskDefArn string, envMap ContainerEnvironmentMap, sess *session.Session) (*ecs.RunTaskOutput, error) {
  to := envMap.ToTaskOverride()
  params := &ecs.RunTaskInput{
    TaskDefinition: aws.String(taskDefArn),
    Cluster: aws.String(clusterName),
    Count: aws.Int64(1),
    Overrides: &to,
  }
  ecsSvc := ecs.New(sess)
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


func RunTask(clusterName string, taskDef string, sess *session.Session) (*ecs.RunTaskOutput, error) {
  env := make(ContainerEnvironmentMap)
  resp, err := RunTaskWithEnv(clusterName, taskDef, env, sess)
  return resp, err
}

func WaitForTaskRunning(clusterName, taskArn string, sess *session.Session) (error) {
  ecsSvc := ecs.New(sess)
  params := &ecs.DescribeTasksInput{
    Cluster: aws.String(clusterName),
    Tasks: []*string{aws.String(taskArn)},
  }
  return ecsSvc.WaitUntilTasksRunning(params)
}

// Should consider returning DTMs for this.
func OnTaskRunning(clusterName, taskArn string, sess *session.Session, do func(*ecs.DescribeTasksOutput, error)) {
    go func() {
      task_params := &ecs.DescribeTasksInput{
        Cluster: aws.String(clusterName),
        Tasks: []*string{aws.String(taskArn)},
      }
      ecsSvc := ecs.New(sess)
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


const (
  ContainerStateRunning = "RUNNING"
  ContainerStatePending = "PENDING"
)

// check for !running and !pending as opposed to just STOPPED.
func ContainerStatusOk(c *ecs.Container) bool {
  return *c.LastStatus == "PENDING" || *c.LastStatus == "RUNNING"
}

