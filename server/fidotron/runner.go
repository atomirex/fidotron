package fidotron

import (
	"fmt"
	"os/exec"
)

type Runner struct {
	broker *Broker
}

type WriterToBroker struct {
	topic  string
	broker *Broker
}

func (w *WriterToBroker) Write(p []byte) (n int, err error) {
	w.broker.Send(w.topic, p)
	return len(p), nil
}

func NewWriterToBroker(broker *Broker, topic string) *WriterToBroker {
	return &WriterToBroker{broker: broker, topic: topic}
}

func NewRunner(b *Broker) *Runner {
	return &Runner{
		broker: b,
	}
}

func (r *Runner) Run(app *App, outtopic string, errtopic string) {
	c := exec.Command(app.Path)

	terminating := make(chan int)

	c.Args = app.Args
	c.Dir = app.Dir
	c.Stdout = NewWriterToBroker(r.broker, outtopic)
	c.Stderr = NewWriterToBroker(r.broker, errtopic)

	go func() {
		err := c.Start()
		if err != nil {
			fmt.Println(err.Error())

			terminating <- -1
			return
		}

		exitcode := 0
		err = c.Wait()
		if err != nil {
			exitcode = err.(*exec.ExitError).ExitCode()
		}
		terminating <- exitcode
	}()

	<-terminating
}
