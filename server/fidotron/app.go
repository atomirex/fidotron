package fidotron

type Application interface {
	Prepare()
	Start()
}

func Run(app Application) {
	// TODO health check endpoint
	// Network init and check
	// Pub/sub maintenance via local broker

	app.Prepare()
	app.Start()
}
