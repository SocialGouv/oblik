package reporting

import "k8s.io/apimachinery/pkg/api/resource"

type UpdateType int

const (
	UpdateTypeCpuRequest UpdateType = iota
	UpdateTypeMemoryRequest
	UpdateTypeCpuLimit
	UpdateTypeMemoryLimit
)

type ResultType int

const (
	ResultTypeSuccess ResultType = iota
	ResultTypeFailed
	ResultTypeDryRun
)

type UpdateResult struct {
	Changes []Change
	Type    ResultType
	Key     string
	Error   error
}

type Change struct {
	Old           resource.Quantity
	New           resource.Quantity
	Type          UpdateType
	ContainerName string
}
