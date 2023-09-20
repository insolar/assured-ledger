package vermap

type Options uint32

const (
	CanUpdate Options = 1 << iota
	CanReplace
	CanDelete
)

const UpdateMask = CanUpdate | CanReplace | CanDelete
