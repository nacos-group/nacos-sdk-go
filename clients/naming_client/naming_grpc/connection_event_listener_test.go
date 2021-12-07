package naming_grpc

import "testing"

func TestRedoSubscribe(t *testing.T) {
	NewConnectionEventListener(new(MockNamingGrpc)).redoSubscribe()
}
