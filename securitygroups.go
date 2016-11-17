package awslib 

import (
  "fmt"
  "strings"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/ec2"
)

func DescribeSecurityGroup(groupId string, sess *session.Session) (*ec2.SecurityGroup, error) {
  ec2Svc := ec2.New(sess)
  params := &ec2.DescribeSecurityGroupsInput {
    GroupIds: []*string{&groupId},
  }
  res, err := ec2Svc.DescribeSecurityGroups(params)

  if err != nil { return nil, err }
  if len(res.SecurityGroups) > 1 { 
    err = fmt.Errorf("Expected 1 security group got %d, only returing the first.", len(res.SecurityGroups))
  }
  return res.SecurityGroups[0], err
}


// TODO: Revist the UserGroupPairs representation, we're leaving out data.
const sgPlaceHolder = "-----"
const sgSeperator = ", "
func IpPermissionStrings(ipp *ec2.IpPermission) (proto, ports, sGroups, ipRanges, prefixes string) {
  proto = *ipp.IpProtocol
  if strings.Compare(proto, "-1") == 0 { proto = "all" }

  var fromPort string
  if ipp.FromPort != nil {
    if *ipp.FromPort == -1 {
      fromPort = "icmpType:all"
    } else {
      fromPort = fmt.Sprintf("%d", *ipp.FromPort)
    }
  }

  var toPort string
  if ipp.ToPort != nil {
    if *ipp.ToPort == -1 {
      toPort = "icmpCode:all"
    } else {
     toPort = fmt.Sprintf("%d", *ipp.ToPort) 
    }
  }

  if (strings.Compare(fromPort, "") == 0) && (strings.Compare(toPort, "") == 0) {
    ports = "all"
  } else {
    ports = fmt.Sprintf("%s-%s", fromPort, toPort)
  }

  sGroups = sgPlaceHolder
  if len(ipp.UserIdGroupPairs) > 0 {
    sGroups = ""
    for _, gp := range ipp.UserIdGroupPairs {
      sGroups += *gp.GroupName + sgSeperator
    }
    sGroups = strings.TrimSuffix(sGroups, sgSeperator)
  }

  ipRanges = sgPlaceHolder
  if len(ipp.IpRanges) > 0 {
    ipRanges = ""
    for _, r := range ipp.IpRanges {
      ipRanges += *r.CidrIp + sgSeperator
    }
    ipRanges = strings.TrimSuffix(ipRanges, sgSeperator)
  }

  prefixes = sgPlaceHolder
  if len(ipp.PrefixListIds) > 0 {
    for _, p := range ipp.PrefixListIds {
      prefixes += *p.PrefixListId + sgSeperator
    }
    prefixes = strings.TrimSuffix(prefixes, sgSeperator)
  }

  return proto, ports, sGroups, ipRanges, prefixes
}