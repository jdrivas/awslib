package awslib

import(
  "sort"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/ecs"
)

func CreateCluster(clusterName string, sess *session.Session) (*ecs.Cluster, error) {
  ecsSvc := ecs.New(sess)
  params := &ecs.CreateClusterInput{
    ClusterName: aws.String(clusterName),
  }
  resp, err := ecsSvc.CreateCluster(params)
  var cluster *ecs.Cluster
  if err == nil {
    cluster = resp.Cluster
  }
  return cluster, err
}

func DeleteCluster(clusterName string, sess *session.Session) (*ecs.Cluster, error) {
  ecsSvc := ecs.New(sess)
  params := &ecs.DeleteClusterInput{
    Cluster: aws.String(clusterName),
  }
  resp, err := ecsSvc.DeleteCluster(params)
  var cluster *ecs.Cluster
  if err == nil {
    cluster = resp.Cluster
  }
  return cluster, err
}

func GetClusters(sess *session.Session) ([]*string, error) {
  ecsSvc := ecs.New(sess)
  arns := make([]*string, 0)
  err := ecsSvc.ListClustersPages(&ecs.ListClustersInput{}, 
    func(page *ecs.ListClustersOutput, lastPage bool) bool {
      arns = append(arns, page.ClusterArns... )
      return true
  })
  return arns, err
}

func DescribeCluster(clusterName string, sess *session.Session) ([]*ecs.Cluster, error) {

  ecsSvc := ecs.New(sess)  
  params := &ecs.DescribeClustersInput {
    Clusters: []*string{aws.String(clusterName),},
  }

  resp, err := ecsSvc.DescribeClusters(params)
  return resp.Clusters, err
}

func GetAllClusterDescriptions(sess *session.Session) (Clusters, error) {
  clusterArns, err := GetClusters(sess)
  if err != nil {return make([]*ecs.Cluster, 0), err}

  ecsSvc := ecs.New(sess)
  params := &ecs.DescribeClustersInput {
    Clusters: clusterArns,
  }
  
  resp, err := ecsSvc.DescribeClusters(params)
  return resp.Clusters, err
}

type Clusters []*ecs.Cluster
type ClusterSortType int
const(
  ByActivity ClusterSortType = iota
  ByReverseActivity
)

func (cs Clusters) Sort(t ClusterSortType) {
  // fmt.Printf("Sorting Clusters.\n)
  switch t  {
  case ByActivity: sort.Sort(clusterByActivity(cs))
  case ByReverseActivity: sort.Sort(rClusterByActivity(cs))
  }
}

// TODO: This is disgusting. There has to be a better way.
type clusterByActivity []*ecs.Cluster
func (cs clusterByActivity) Len() int { return len(cs) }
func (cs clusterByActivity) Swap(i,j int) { cs[i], cs[j] = cs[j], cs[i] }
func (cs clusterByActivity) Less (i, j int) bool {
  if cs[i].Status == nil && cs[j].Status == nil {return *cs[i].ClusterArn < *cs[j].ClusterArn}
  if (cs[i].Status == nil) { return true}
  if (cs[j].Status == nil ) { return false }

  if *cs[i].Status != *cs[j].Status { return *cs[i].Status < *cs[j].Status }

  if *cs[i].RunningTasksCount != *cs[j].RunningTasksCount { return *cs[i].RunningTasksCount < *cs[j].RunningTasksCount }
  return *cs[i].PendingTasksCount < *cs[j].PendingTasksCount
}

type rClusterByActivity []*ecs.Cluster
func (cs rClusterByActivity) Len() int { return len(cs) }
func (cs rClusterByActivity) Swap(i, j int) { cs[i], cs[j] = cs[j], cs[i] }
func (cs rClusterByActivity) Less (j, i int) bool {
  if cs[i].Status == nil && cs[j].Status == nil {return *cs[i].ClusterArn < *cs[j].ClusterArn}
  if (cs[i].Status == nil) { return true}
  if (cs[j].Status == nil ) { return false }
  if *cs[i].Status != *cs[j].Status { return *cs[i].Status < *cs[j].Status }

  if *cs[i].RunningTasksCount != *cs[j].RunningTasksCount { return *cs[i].RunningTasksCount < *cs[j].RunningTasksCount }
  return *cs[i].PendingTasksCount < *cs[j].PendingTasksCount
}

type ClusterCache map[string]bool


func (cc *ClusterCache) Update(sess *session.Session) (error) {
  clusterArns, err := GetClusters(sess)
  if err != nil { return err }

  cc.Empty()
  for _, a := range clusterArns {
    n := ShortArnString(a)
    (*cc)[n] = true
  }
  return err
}

func (cc *ClusterCache) Empty() {
  for k, _ := range *cc {
    delete(*cc,k)
  }
}

func (cc *ClusterCache) Contains(v string, sess *session.Session) (contains bool, err error) {
  if (*cc)[v] { return true, nil }

  err = cc.Update(sess)
  if err != nil { return false, err }

  if (*cc)[v] { return true, nil }
  return false, nil
}
