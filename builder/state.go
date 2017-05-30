package builder

import (
	"sync"

	"github.com/Sirupsen/logrus"
)

// Будем хранить в структуре State все данные, необходимые для конкурентного выполнения.
// 1. Задаем канал taskPool такой емкости, сколько задач одновременно мы готовы обрабатывать,
// и запускаем workers для обработки этого канала.
// 2. Также задаем taskQueue - список задач, поступивших на обработку, и inProgress - список обрабатываемых
// в настоящий момент пользователей. taskQueue - это очередь на добавление в taskPool, а inProgress - это задачи,
// которые в настоящий момент находятся в taskPool.
// Для этих списков ставим мьютекс для того, чтобы конкурентно брать задачи и учитывать пользователей.
// 3. processPool: Одновременная обработка одного и того же репозитория для одного и того же пользователя невозможна
// (нельзя обработать параллельно разные коммиты), поэтому в случае, если на обработку попадает задача
// для такого пользователя и репозитория, который уже обрабатывается, эту задачу перенесем в конец списка задач.

// ToDo: add graceful shutdown!
// ToDo: add max execution time per task!!!

// Task represents a Task for CI/CD.
type Task struct {
	callback Callback
	id       string
	task     string
	prefix   string
	user     string
	repo     string
	commit   string
}

// NewTask creates an instance of a task.
func NewTask(callback Callback, id, task, prefix, user, repo, commit string) Task {
	return Task{
		callback: callback,
		id:       id,
		task:     task,
		prefix:   prefix,
		user:     user,
		repo:     repo,
		commit:   commit,
	}
}

// Processor is a function to process tasks
type Processor func(logger logrus.FieldLogger, task Task)

// Callback is a function to update information about current task state
type Callback func(status string, description string)

// State is a state of building process, it must live during the service is working
type State struct {
	logger    logrus.FieldLogger
	processor Processor

	taskPool chan Task

	mutex      *sync.RWMutex
	inProgress map[string]Task
	taskQueue  []Task
}

// NewState creates new State instance with the parameters:
// - processor to process a task (use builder.Process for it)
// - logger to log important information
// - special channel to be able to process only workers goroutines in the same time
// - list of processing tasks and a mutex to deal with them
// - list of 'to do' tasks and a mutex to deal with them
// - shutdown channel to mark that service is not available for tasks anymore
func NewState(processor Processor, logger logrus.FieldLogger, maxWorkers int) *State {
	state := &State{
		processor:  processor,
		logger:     logger,
		taskPool:   make(chan Task, maxWorkers),
		mutex:      &sync.RWMutex{},
		inProgress: make(map[string]Task),
	}

	for i := 0; i < maxWorkers; i++ {
		go state.worker(i)
	}

	go state.processPool()

	return state
}

func (state *State) GetTasks() ([]string, []string) {
	var queue []string
	var current []string

	state.mutex.Lock()
	defer state.mutex.Unlock()

	for _, task := range state.taskQueue {
		queue = append(queue, task.user+"/"+task.repo+": "+task.id)
	}

	for user, task := range state.inProgress {
		current = append(current, user+"/"+task.repo+": "+task.id)
	}

	return queue, current
}

// AddTask add Task to the pool.
// ToDo: if shutdown command is sent, the Task will nod be added and the function will return an error.
func (state *State) AddTask(callback Callback, id, task, prefix, user, repo, commit string) error {
	logger := state.logger.WithField("task_id", id)
	logger.Infof("Add task %s...", id)

	t := Task{
		callback: callback,
		id:       id,
		task:     task,
		prefix:   prefix,
		user:     user,
		repo:     repo,
		commit:   commit,
	}

	state.mutex.Lock()
	state.taskQueue = append(state.taskQueue, t)
	state.mutex.Unlock()

	logger.Info("Task added.")

	return nil
}

func (state *State) worker(i int) {
	for {
		select {
		case t := <-state.taskPool:
			logger := state.logger.WithField("task_id", t.id)
			logger.Infof("Worker #%d processing task %s...", i, t.id)

			state.processor(state.logger, t)

			state.mutex.Lock()
			delete(state.inProgress, t.user)
			logger.Infof("Worker #%d processed task %s.", i, t.id)
			state.mutex.Unlock()
		}
	}
}

// processPool gets Task from the pool.
// You can call this function concurrently, state.workers parameter decides if it is possible to add one more worker.
func (state *State) processPool() {
	for {
		if state.queueLen() == 0 {
			continue
		}

		state.mutex.Lock()

		t := state.taskQueue[0]

		logger := state.logger.WithField("task_id", t.id)
		logger.Debugf("Task %s is getting from the queue...", t.id)

		inProgress, ok := state.inProgress[t.user]

		// if this user is already in progress, move him to the end of the queue
		// (couldn't process the same user at the same time)
		addToPool := false
		if ok && (inProgress.repo == t.repo) {
			state.taskQueue = append(state.taskQueue, t)
			logger.Debugf("Task %s moved back to the queue.", t.id)
		} else {
			// otherwise mark user as 'in progress'
			state.inProgress[t.user] = t
			addToPool = true
			logger.Debugf("Task %s is moving to the pool and is going to be processed...", t.id)
		}

		state.taskQueue = state.taskQueue[1:]

		state.mutex.Unlock()

		if addToPool {
			state.taskPool <- t
		}
	}
}

func (state *State) queueLen() int {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	return len(state.taskQueue)
}

func (state *State) queuesEmpty() bool {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	return len(state.taskQueue) == 0 && len(state.inProgress) == 0
}
