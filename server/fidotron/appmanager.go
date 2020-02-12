package fidotron

type AppManager struct {
	apps map[string]*App
}

type App struct {
	Name string
	Args []string
	Dir  string
	Path string
}

func NewAppManager() *AppManager {
	return &AppManager{
		apps: make(map[string]*App),
	}
}

func (am *AppManager) Add(app *App) {
	am.apps[app.Name] = app
}

func (am *AppManager) App(name string) *App {
	return am.apps[name]
}
