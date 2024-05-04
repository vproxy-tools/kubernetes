package shared

import "k8s.io/apiserver/pkg/server"

var LogIsInitiated = false
var signalHandler <-chan struct{} = nil

func SetupSignalHandler() <-chan struct{} {
	if signalHandler == nil {
		signalHandler = server.SetupSignalHandler()
	}
	return signalHandler
}
