package awslib

import(
  "io"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/ecs"
  "github.com/aws/aws-sdk-go/private/protocol/json/jsonutil"
  // "github.com/Sirupsen/logrus"
)

// Lists ACTIVE families of Task Definitions, returning a colelctio of td arns.
func ListTaskDefinitionFamilies(sess *session.Session) ([]*string, error) {
  ecsSvc := ecs.New(sess)
  params := &ecs.ListTaskDefinitionFamiliesInput{
    Status: aws.String("ACTIVE"),
  }
  results := make([]*string,0)
  err := ecsSvc.ListTaskDefinitionFamiliesPages(params,
    func(p *ecs.ListTaskDefinitionFamiliesOutput, lastPage bool) (bool) {
      results = append(results, p.Families...)
      return lastPage
  })
  return results, err
}

// returns a collection of all registred task definition arns.
func ListTaskDefinitions(ecs_svc *ecs.ECS) ([]*string, error) {
  params := &ecs.ListTaskDefinitionsInput{
    MaxResults: aws.Int64(100),
  }
  resp, err := ecs_svc.ListTaskDefinitions(params)
  return resp.TaskDefinitionArns, err
}

// func GetTaskDefinition(taskDefinitionArn string, ecs_svc *ecs.ECS) (*ecs.TaskDefinition, error) {
func GetTaskDefinition(taskDefinitionArn string, sess *session.Session) (*ecs.TaskDefinition, error) {
  ecsSvc := ecs.New(sess)
  params := &ecs.DescribeTaskDefinitionInput {
    TaskDefinition: aws.String(taskDefinitionArn),
  }
  resp, err := ecsSvc.DescribeTaskDefinition(params)
  return resp.TaskDefinition, err
}

// TODO: Sigh ... need to go build the basic Collection functions.
func GetContainerDefinition(containerName string, td *ecs.TaskDefinition) (cd *ecs.ContainerDefinition, ok bool) {
  for _, d  := range td.ContainerDefinitions {
    if *d.Name == containerName {
      ok = true
      cd = d
      break
    }
  }
  return cd, ok
}

// TODO: This relies on an unsupported JSON unmarshalling interface in the aws go-sdk.
// This could stop working.
func RegisterTaskDefinitionWithJSON(json io.Reader, sess *session.Session) (*ecs.RegisterTaskDefinitionOutput, error) {
  var tdi ecs.RegisterTaskDefinitionInput
  err := jsonutil.UnmarshalJSON(&tdi, json)
  if err != nil { return nil, err}
  log.Debug(nil, "RegisterTaskDefinition: Decoded JSON stream.")

  ecsSvc := ecs.New(sess)
  resp, err := ecsSvc.RegisterTaskDefinition(&tdi)
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