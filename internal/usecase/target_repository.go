package usecase

// ITargetRepository combines all repository operations required to manage targets.
type ITargetRepository interface {
	IAddTargetRepository
	IGetTargetRepository
	IListTargetsRepository
	IDeleteTargetRepository
	IUpdateTargetRepository
}
