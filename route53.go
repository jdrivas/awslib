package awslib

import(
  "fmt"
  "strings"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  // "github.com/aws/aws-sdk-go/service/ec2"
  "github.com/aws/aws-sdk-go/service/route53"
  "github.com/Sirupsen/logrus"
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

// This deletes the DNS record for the FQDN.
func DetachFromDNS(ip, fqdn, comment string, ttl int64, sess *session.Session) (*route53.ChangeInfo, error) {

  zone, err := GetHostedZone(fqdn, sess)
  if err != nil { return nil, err }

  newaddr := strings.ToLower(fqdn) + "."
  r53Svc := route53.New(sess)
  params := &route53.ChangeResourceRecordSetsInput{
    HostedZoneId: zone.Id,
    ChangeBatch: &route53.ChangeBatch{
      Comment: aws.String(comment),
      Changes: []*route53.Change{
        {
          Action: aws.String("DELETE"),
          ResourceRecordSet: &route53.ResourceRecordSet{
            Name: aws.String(newaddr),
            Type: aws.String("A"),
            ResourceRecords: []*route53.ResourceRecord{
              { Value: aws.String(ip) },
            },
            TTL: aws.Int64(ttl),
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

// returns recrods assocated with the baseFQDN provided.
func ListDNSRecords(baseFQDN string, sess *session.Session) ([]*route53.ResourceRecordSet, error) {
  hz, err := GetHostedZone(baseFQDN, sess)
  if err != nil { return nil, err }

  r53Svc := route53.New(sess)
  reqIter, totalCount, keptRecords := 0,0,0
  f := logrus.Fields{"baseFQDB": baseFQDN, "totalCount": totalCount, "reqIter": reqIter, "keptRecords": keptRecords}
  records := make([]*route53.ResourceRecordSet,0)
  params := &route53.ListResourceRecordSetsInput{
    HostedZoneId: hz.Id,
    StartRecordName: aws.String(baseFQDN),
    // StartRecordType: aws.String("A")
  }
  err = r53Svc.ListResourceRecordSetsPages(params, 
    func(page *route53.ListResourceRecordSetsOutput, lastPage bool) bool {
      for _, r := range page.ResourceRecordSets {
        if strings.Contains(*r.Name, baseFQDN) {
          records = append(records, r)
          keptRecords++
        }
        totalCount++
      }
      reqIter++
      f["reqIter"] = reqIter
      f["totalCount"] = totalCount
      f["keptRecords"] = keptRecords
      log.Debug(f, "Received records for DNS lookup.")
      return true
    })
  return records, err
}

// Pull out the TLD from the fqdn.
func getZoneString(fqdn string) (string, bool) {
  dn := strings.Split(fqdn, ".")
  l := len(dn)
  if l < 2 || dn[l-2] == "" { return "", false }
  return dn[l-2] + "." + dn[l-1], true
}

func OnDNSChangeSynched(changeId *string, sess *session.Session, 
  do func(*route53.ChangeInfo, error)) {
  go func() {
    r53Svc := route53.New(sess)
    param := &route53.GetChangeInput{
      Id: changeId,
    }
    err := r53Svc.WaitUntilResourceRecordSetsChanged(param)
    resp, err2 := r53Svc.GetChange(param)
    if err == nil { err = err2}
    do(resp.ChangeInfo, err)
  }()
}

