package awslib

import(
  "fmt"
  "strings"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  // "github.com/aws/aws-sdk-go/service/ec2"
  "github.com/aws/aws-sdk-go/service/route53"
)

// TODO: This is very basic. At some point HealthChecks and GeoLocation may be a good idea.
// We create an A reocrd mapping the FQDN.
func AttachIpToDNS(ip, fqdn, comment string, ttl int64, sess *session.Session) (*route53.ChangeInfo, error) {

  zone, err  := GetHostedZone(fqdn, sess)
  if err != nil { return nil, err }

  // TODO: make this more robust in the face of varying input.
  newaddr := strings.ToLower(fqdn) + "."
  r53Svc := route53.New(sess)
  params := &route53.ChangeResourceRecordSetsInput{
    HostedZoneId: zone.Id,
    ChangeBatch: &route53.ChangeBatch{
      Comment: aws.String(comment),
      Changes: []*route53.Change{
        {
          Action: aws.String("UPSERT"),
          ResourceRecordSet: &route53.ResourceRecordSet{
            Name: aws.String(newaddr),
            Type: aws.String("A"),
            ResourceRecords: []*route53.ResourceRecord{
              { Value: aws.String(ip) },
            },
            TTL: aws.Int64(ttl), // this seems to be required!
          },
        },
      },
    },
  }
  resp, err := r53Svc.ChangeResourceRecordSets(params)
  return resp.ChangeInfo, err
}

// Get the hosted zone for the fqdn
func GetHostedZone(fqdn string, sess *session.Session) (*route53.HostedZone, error) {

  zone, ok := getZoneString(fqdn)
  if !ok { return nil, fmt.Errorf("GetHostedZone: doesn't seem to be a FQDN: %s", fqdn) }

  param := &route53.ListHostedZonesByNameInput{
    DNSName: aws.String(zone),
  }
  r53Svc := route53.New(sess)
  resp, err := r53Svc.ListHostedZonesByName(param)
  if err != nil { return nil, err }

  // Can't search in the above with the final '.', but 
  // the response will include them.
  zone = zone + "."
  var hzone *route53.HostedZone
  for _, hz := range resp.HostedZones {
    if *hz.Name == zone {
      hzone = hz
      break
    }
  }
  if hzone == nil {
    err = fmt.Errorf("Couldn't find a hosted zone for %s (%s).", fqdn, zone)
  }
  return hzone, err
}

// Pull out the TLD from the fqdn.
func getZoneString(fqdn string) (string, bool) {
  dn := strings.Split(fqdn, ".")
  l := len(dn)
  if l < 2 || dn[l-2] == "" { return "", false }
  return dn[l-2] + "." + dn[l-1], true
}



