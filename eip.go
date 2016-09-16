package awslib

import(

  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/ec2"
)


func GetNewEIP(sess *session.Session) (*ec2.AllocateAddressOutput, error) {
  ec2Svc := ec2.New(sess)
  param := &ec2.AllocateAddressInput{
    Domain: aws.String("vpc"),
  }
  return ec2Svc.AllocateAddress(param)
}

func AssociateEIP(allocationId, instanceId *string, sess *session.Session) (*string, error) {
  ec2Svc := ec2.New(sess)
  param := &ec2.AssociateAddressInput{
    AllocationId: allocationId,
    InstanceId: instanceId,
    // PrivateIdAddress: // We'll use the default for now.
  }
  resp, err := ec2Svc.AssociateAddress(param)
  return resp.AssociationId, err
}