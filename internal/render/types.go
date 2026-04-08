package render

// OutputOp represents a write operation to stdout
type OutputOp struct {
	Data []byte
}

// OutputChan is the interface for writing output
type OutputChan interface {
	WriteOp(op OutputOp)
}
