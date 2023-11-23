package Rpc

var isStopRequested bool = false

type ForkliftControlRpc struct {
}

func NewControlRpc() *ForkliftControlRpc {
	var srv = ForkliftControlRpc{}
	return &srv
}

func (server *ForkliftControlRpc) IsStopRequested() bool {
	return isStopRequested
}

func (server *ForkliftControlRpc) Stop(_ *int, _ *bool) error {
	isStopRequested = true
	return nil
}
