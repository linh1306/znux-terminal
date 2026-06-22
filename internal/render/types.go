package render

// OutputOpKind identifies non-byte-write operations for the stdout owner.
type OutputOpKind int

const (
	OutputOpWrite OutputOpKind = iota
	OutputOpPTYWrite
	OutputOpClearCurrent
	OutputOpHistoryPrev
	OutputOpHistoryNext
)

// OutputOp represents a write operation to stdout
type OutputOp struct {
	Kind OutputOpKind
	Data []byte
}

// OutputChan is the interface for writing output
type OutputChan interface {
	WriteOp(op OutputOp)
}
