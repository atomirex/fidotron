package fidotron

import "io"

const (
	Running = iota
	Failed
)

type ServiceStatus int

type ServiceState interface {
	Status() ServiceStatus
	Load(io.Reader)
	Save(io.Writer)
}

type ServiceMessage struct {
	What    int
	Arg1    int
	Arg2    int
	Payload interface{}
}

type ServiceContext interface {
	Trace()
	Log()
	Send()
}

type Service interface {
	Handle(context ServiceContext, state ServiceState, message ServiceMessage) ServiceState
}
