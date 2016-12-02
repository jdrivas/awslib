package awslib

import(
  "fmt"
  // "time"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/ecs"
  // "github.com/aws/aws-sdk-go/service/ec2"
  // "github.com/Sirupsen/logrus"
)

// Get a list of all defined service arns for a cluster.
func ListServices(clusterName string, sess *session.Session) (services []*string, err error) {

  ecsSvc := ecs.New(sess)
  params := &ecs.ListServicesInput{
    Cluster: aws.String(clusterName),
  }
  services = make([]*string,0)
  err = ecsSvc.ListServicesPages(params, func(page *ecs.ListServicesOutput, lastPage bool) (bool) {
    services = append(services, page.ServiceArns...)
    return true
  })

  return services, err
}

func DescribeServices(clusterName string, sess *session.Session) (services  []*ecs.Service, failures []*ecs.Failure, err error) {

  serviceArns, err := ListServices(clusterName, sess)
  if err != nil || len(serviceArns) == 0 { return services, failures, err }

  ecsSvc := ecs.New(sess)
  params := &ecs.DescribeServicesInput {
    Cluster: aws.String(clusterName),
    Services: serviceArns,
  }
  res, err := ecsSvc.DescribeServices(params)

  return res.Services, res.Failures, err
}

func DescribeService(serviceName, clusterName string, 
  sess *session.Session) (service *ecs.Service, failures []*ecs.Failure, err error) {

  ecsSvc := ecs.New(sess)
  params := &ecs.DescribeServicesInput{
    Cluster: aws.String(clusterName),
    Services: []*string{aws.String(serviceName)},
  }
  res, err := ecsSvc.DescribeServices(params)

  if err == nil {
    switch {
    case len(res.Services) == 1:
      service = res.Services[0]
    case len(res.Services) == 0:
      err = fmt.Errorf("Received no services.")
    case len(res.Services) > 1:
      ss := ""
      for i, s := range res.Services {
        ss += fmt.Sprintf("%d. %#v\n", i+1, *s)
      }
      err = fmt.Errorf("Error received more than one service in response: %s", ss)
    }
  }

  return service, res.Failures, err
}


// Create a service without a LoadBlancer.
func CreateService(serviceName, clusterName , taskDefinitionArn string, 
  instanceCount int64, sess *session.Session) (s *ecs.Service, err error) {

  ecsSvc := ecs.New(sess)
  params := &ecs.CreateServiceInput {
    ServiceName: aws.String(serviceName),
    TaskDefinition: aws.String(taskDefinitionArn),
    Cluster: aws.String(clusterName),
    DesiredCount: aws.Int64(instanceCount),
  }

  res, err := ecsSvc.CreateService(params)
  if err == nil { s = res.Service }

  return s, err
}

// Update a service with taskDefinition and instanceCount.
func UpdateService(serviceName, clusterName, taskDefinitionArn string, 
  instanceCount int64, sess *session.Session) (s *ecs.Service, err error) {

  ecsSvc := ecs.New(sess)
  params := &ecs.UpdateServiceInput{
    Service: aws.String(serviceName),
    TaskDefinition: aws.String(taskDefinitionArn),
    Cluster: aws.String(clusterName),
    DesiredCount: aws.Int64(instanceCount),
  }

  res, err := ecsSvc.UpdateService(params)
  if err == nil { s = res.Service }

  return s, err
}

// This takes a somewhat draconian approach to restart. It sets the desired count to 0
// then resets it to it's original value. Note: in the process we h ave to set the MinimumHealthyPrecent to
// something less that 100 so that the last task gets killed.
// This whole thing is necessary when you need to update some not TaskDefinition related feature to the
// system (e.g. configuration files, images etc.). It would be better if we could instruct the system to 
// just do it's normal update on command without thinking for us if it's needed or not.
func RestartService(serviceName, clusterName string, sess *session.Session, cb func(*ecs.Service, error)) (err error) {

  sOrig, failures, err := DescribeService(serviceName, clusterName, sess)
  if err != nil { return err }
  if len(failures) > 0 { return fmt.Errorf("Failed when obtaining service description: %#v.", failures) }


  oDCnt := *sOrig.DesiredCount
  oDConfig := sOrig.DeploymentConfiguration
  dConfig := *oDConfig
  if *dConfig.MinimumHealthyPercent >= 100 {
    *dConfig.MinimumHealthyPercent = 49
  }

  // Stop services, buy setting desired count to 0.
  ecsSvc := ecs.New(sess)
  params := &ecs.UpdateServiceInput{
    Service: aws.String(serviceName),
    Cluster: aws.String(clusterName),
    DesiredCount: aws.Int64(0),           // this is the only way I know to reliablly stop the service.
    DeploymentConfiguration: &dConfig,
  }
  res, err := ecsSvc.UpdateService(params)

  // Wait to stabilize and use the callback when you do.
  go func() {
    waitParams := &ecs.DescribeServicesInput{
      Services: []*string{aws.String(serviceName)},
      Cluster: aws.String(clusterName),
    }
    err = ecsSvc.WaitUntilServicesStable(waitParams)
    if err != nil { cb(nil, fmt.Errorf("Restart service failure setting DesiredCount to 0: %s", err)) }

    // Restart the service, don't forget to reset the minimum.
    s  := res.Service
    tdFamily := TaskDefinitionFamily(sOrig.TaskDefinition)
    params.TaskDefinition = aws.String(tdFamily)
    params.DesiredCount = aws.Int64(oDCnt)
    params.DeploymentConfiguration = oDConfig
    // params.DeploymentConfiguration.MinimumHealthyPercent = oDConfig.MinimumHealthyPercent
    // *params.DeploymentConfiguration.MinimumHealthyPercent = 
    nRes, err := ecsSvc.UpdateService(params)
    if err == nil { s = nRes.Service }

    cb(s, err)
  }()

  return err
}

func UpdateServiceDesiredCount(serviceName, clusterName string,
  instanceCount int64, sess *session.Session) (s *ecs.Service, err error) {

  ecsSvc := ecs.New(sess)
  params := &ecs.UpdateServiceInput {
    Service: aws.String(serviceName),
    Cluster: aws.String(clusterName),
    DesiredCount: aws.Int64(instanceCount),
  }

  res, err := ecsSvc.UpdateService(params)
  if err == nil { s = res.Service}

  return s, err
}


// TODO: Delete will fail if the status of the service is Primary and the desired count is > 0. Determine
// if we should note this and fail to delete, or update the service to 0 and then delete or what?

// Delete a service.
// Delete will fail if primary deployment is > 0.
func DeleteService(serviceName, clusterName string, sess *session.Session) (s *ecs.Service, err error) {

  ecsSvc := ecs.New(sess)
  params := &ecs.DeleteServiceInput {
    Service: aws.String(serviceName),
    Cluster: aws.String(clusterName),
  }

  res, err := ecsSvc.DeleteService(params)
  if err == nil { s = res.Service }

  return s, err
}


// Registers func() to be fired when the Service becomes stable.
// Used often after a create to fire an update on ready.
func OnServiceStable(serviceName, clusterName string, sess *session.Session, do func(error)) {
  ecsSvc := ecs.New(sess)
  go func() {
    params := &ecs.DescribeServicesInput{
      Services: []*string{aws.String(serviceName)},
      Cluster: aws.String(clusterName),
    }
    err := ecsSvc.WaitUntilServicesStable(params)
    do(err)
  }()
}

// Registers func() to be fired when the Service becomes inactive.
// Used often after a create to fire an update on ready.
func OnServiceInactive(serviceName, clusterName string, sess *session.Session, do func(error)) {
  ecsSvc := ecs.New(sess)
  go func() {
    params := &ecs.DescribeServicesInput{
      Services: []*string{aws.String(serviceName)},
      Cluster: aws.String(clusterName),
    }
    err := ecsSvc.WaitUntilServicesInactive(params)
    do(err)
  }()
}
