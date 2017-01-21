package awslib 

import (
  "fmt"
  "errors"
  "strconv"
  "time"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/ecs"
  "github.com/aws/aws-sdk-go/service/ec2"
  // "github.com/Sirupsen/logrus"
)

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

func (ci *ContainerInstance) RegisteredResources() (ResourceMap) {
  return collectResources(ci.Instance.RegisteredResources)
}

func (ci *ContainerInstance) RemainingResources() (ResourceMap) {
  return collectResources(ci.Instance.RemainingResources)
}

// TODO: THIS BADLY NEEDS REFACTORING ... the next two API points are too close.
// func GetContainerInstanceDescriptions(clusterName string, sess *session.Session) (cis []*ecs.ContainerInstance, cifs []*ecs.Failure, err error) {
//   instanceArns, err := GetContainerInstances(clusterName, sess)
//   if err != nil || len(instanceArns) <= 0 { return cis, cifs, err }

//   ecsSvc := ecs.New(sess)
//   params := &ecs.DescribeContainerInstancesInput {
//     ContainerInstances: instanceArns,
//     Cluster: aws.String(clusterName),
//   }
//   dcio, err := ecsSvc.DescribeContainerInstances(params)
//   return dcio.ContainerInstances, dcio.Failures, err
// }

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

// These are the constants for the known keys for resources.

const(
  MEMORY = "MEMORY"
  CPU = "CPU"
)

// This will add a new resoruce to the map,
// it will aggregate values if add a resource
// with the same name as one already in the map.
func (rMap ResourceMap) Add(r *ecs.Resource) {

  // Check to see if we've already got the named reesource
  newR, ok := rMap[*r.Name]
  n := *r.Name
  t := *r.Type
  if !ok { // create a new one.
    newR = new(ecs.Resource)
    rMap[*r.Name] = newR
    newR.Name = &n
    newR.Type = &t
    switch *r.Type {
      case "INTEGER":
        newR.IntegerValue = new(int64)
      case "LONG": 
        newR.LongValue = new(int64)
      case "DOUBLE": 
        newR.DoubleValue = new(float64)
      case "STRINGSET": 
        newR.StringSetValue = make([]*string, 0, len(r.StringSetValue))
    }
  }

  // Now set or aggregate the value
  switch *r.Type {
    case "INTEGER":
      *newR.IntegerValue += *r.IntegerValue
    case "LONG": 
      *newR.LongValue += *r.LongValue
    case "DOUBLE": 
      *newR.DoubleValue +=  *r.DoubleValue
    case "STRINGSET":
      newR.StringSetValue = append(newR.StringSetValue, r.StringSetValue...)
  }

}

func collectResources(rs []*ecs.Resource) (ResourceMap) {
  // cs := make([]*ecs.Resource, 0)

  // csMap := make(map[string]*ecs.Resource,0)
  csMap := make(ResourceMap, 0)
  for _, r := range rs {
    csMap.Add(r)
  }
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

func TerminateContainerInstance(clusterName string, containerArn string, sess *session.Session) (resp *ec2.TerminateInstancesOutput, err error) {

  ecsSvc := ecs.New(sess)

  // Need to get the container instance description in order to get the ec2-instanceID.
  params := &ecs.DescribeContainerInstancesInput{
    ContainerInstances: []*string{aws.String(containerArn)},
    Cluster: aws.String(clusterName),
  }
  dci_resp, err := ecsSvc.DescribeContainerInstances(params)
  if err != nil {
    return nil, err
  }

  instanceId := getInstanceId(dci_resp.ContainerInstances, containerArn)
  if instanceId == nil {
    errMessage := fmt.Sprintf("TerminateContainerInstance: Can't find the Ec2 Instance ID for container arn: %s", containerArn)
    err = errors.New(errMessage)
    resp = nil
  } else {
   resp, err = TerminateInstance(instanceId, sess)
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

