package awslib

import(
  // "fmt"
  "sort"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/ecr"
)
type RepositoryList []*ecr.Repository
func GetRepositories(sess *session.Session) (repos RepositoryList, err error) {
  ecrSvc := ecr.New(sess)
  repos = make(RepositoryList, 0)
  err = ecrSvc.DescribeRepositoriesPages(&ecr.DescribeRepositoriesInput{},
    func(page *ecr.DescribeRepositoriesOutput, lastPage bool) (bool) {
      repos = append(repos, page.Repositories...)  
      return true
    })
  return repos, err
}

// sorting Interface
type repoSort struct {
  repos RepositoryList
  less func( iR, jR *ecr.Repository) (bool)
}
func (s repoSort) Len() int { return len(s.repos) }
func (s repoSort) Swap(i, j int) { s.repos[i], s.repos[j] = s.repos[j], s.repos[i] }
func (s repoSort) Less(i, j int) (bool) { return s.less(s.repos[i], s.repos[j]) }

func ByRepoName(repos RepositoryList) (repoSort) {
  return repoSort{
    repos: repos,
    less: func( ir, jr *ecr.Repository) (bool) { return *ir.RepositoryName < *jr.RepositoryName },
  }
}

func ByRepoCreatedAt(repos RepositoryList) (repoSort) {
  return repoSort{
    repos: repos,
    less: func( ir, jr *ecr.Repository) (bool) { return ir.CreatedAt.Before(*jr.CreatedAt) },
  }
}

func ByRepoLastUpdate(repos RepositoryList) (repoSort) {
  return repoSort{
    repos: repos,
    less: func( ir, jr *ecr.Repository) (bool) { return ir.CreatedAt.Before(*jr.CreatedAt) },
  }
}


type ImageDetailList []*ecr.ImageDetail
func GetImages(repositoryName string, sess *session.Session) (ids ImageDetailList, err error) {
  ecrSvc := ecr.New(sess)
  ids = make([]*ecr.ImageDetail, 0)
  err = ecrSvc.DescribeImagesPages(
    &ecr.DescribeImagesInput{
      RepositoryName: &repositoryName,
    }, 
    func(page *ecr.DescribeImagesOutput, lastPage bool) bool {
      ids = append(ids, page.ImageDetails...)
      return true
  })
  return ids, err
}

// returns all the images indexed by repository name. Each list of images is sorted
// by reverse PushedAt time. So the the first image in the list is the most recently
// pushed.
func GetAllImages(sess *session.Session) (imageMap map[string]ImageDetailList, err error) {
  repos, err := GetRepositories(sess)
  if err != nil { return imageMap, err}

  imageMap = make(map[string]ImageDetailList, len(repos))
  for _, r := range repos {
    idl, err := GetImages(*r.RepositoryName, sess)
    if err != nil { return imageMap, err }
    sort.Sort(sort.Reverse(ByPushedAt(idl)))
    imageMap[*r.RepositoryName] = idl
  }
  return imageMap, err
}

// Sorting interface
type imageSort struct {
  ids ImageDetailList
  less func(idI, idJ *ecr.ImageDetail) (bool)
}
func (s imageSort) Len() int { return len(s.ids) }
func (s imageSort) Swap(i, j int) { s.ids[i], s.ids[j] = s.ids[j], s.ids[i] }
func (s imageSort) Less(i, j int) bool { return s.less(s.ids[i], s.ids[j]) }

func ByPushedAt(ids ImageDetailList) (imageSort) {
  return imageSort{
    ids: ids,
    less: func(idI, idJ *ecr.ImageDetail) (bool) {
      return idI.ImagePushedAt.Before(*idJ.ImagePushedAt)
    },
  }
}

