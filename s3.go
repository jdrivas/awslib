package awslib

import(
  "fmt"
  // "github.com/aws/aws-sdk-go/service/s3"
)

// Suitable for constructing S3 object URIs
// like: S3_BASE_URL + "/" + bucket_name + "/" + key
const S3_BASE_URL = "https://s3.amazonws.com"

// Don't include forward-slash in the bucket name
// or at the head of the key (e.g. not /keyname or /bucket-name/)
func S3URI(bucketName, key string) (string) {
  return fmt.Sprintf("%s/%s/%s", S3_BASE_URL, bucketName, key)
}
