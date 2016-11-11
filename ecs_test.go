package awslib

import(
  "strconv"
  "testing"
  "github.com/aws/aws-sdk-go/service/ecs"
  "github.com/stretchr/testify/assert"
)

var CPU = "CPU"
var MEM = "MEM"
var ITYPE = "INTEGER"
var INT1024 = int64(1024)
var INT128 = int64(128)

var MEMR = &ecs.Resource{
  Name: &MEM,
  Type: &ITYPE,
  IntegerValue: &INT1024,
}

var CPUR = &ecs.Resource{
  Name: &CPU,
  Type: &ITYPE,
  IntegerValue: &INT128,
}

func TestResourceMap(t *testing.T)  {
  rm := make(ResourceMap,0)
  rm.Add(MEMR)
  rm.Add(CPUR)
  assert.Equal(t, strconv.FormatInt(INT1024,10), rm.StringFor(MEM))
  assert.Equal(t, strconv.FormatInt(INT128,10), rm.StringFor(CPU))
}

func TestRMAggregation(t *testing.T) {
  rm := make(ResourceMap, 0)
  rm.Add(MEMR)
  rm.Add(MEMR)
  rm.Add(CPUR)
  assert.Equal(t, strconv.FormatInt(INT1024*2,10), rm.StringFor(MEM))
  assert.Equal(t, strconv.FormatInt(INT128,10), rm.StringFor(CPU))
}
