package support

type Worker struct {
	Name string
	Cron string // cron expression, empty = standalone
	Run  func()
}

var workers []*Worker

func RegisterWorker(w *Worker) {
	workers = append(workers, w)
}

func GetWorkers() []*Worker {
	return workers
}
