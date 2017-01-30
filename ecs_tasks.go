package awslib

import (
  "fmt"
  "time"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/ecs"
)

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
  Task *ecs.Task          `json: "task" locationName:"task"`
  Failure *ecs.Failure    `json: "failure" locationName:"failure"`
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
    // ctMap[*task.TaskArn] =  ct
    ctMap[ShortArnString(task.TaskArn)] =  ct
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
