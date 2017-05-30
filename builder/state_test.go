package builder

import (
	"sync"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
)

func prepareState(mux *sync.RWMutex, completed map[string]string, maxWorkers int) *State {
	processor := func(logger logrus.FieldLogger, task Task) {
		time.Sleep(100 * time.Millisecond)
	}

	// callback "counts" all tasks which were processed
	callback := func(logger logrus.FieldLogger, task Task) {
		mux.Lock()
		completed[task.id] = task.id
		mux.Unlock()
	}
	state := NewState(processor, callback, logrus.WithField("max_workers", maxWorkers), maxWorkers)
	return state
}

// Please, run tests with race detector: go test -v -race
func TestAddTask(t *testing.T) {
	maxWorkers := []int{1, 2, 5, 10, 20}

	for _, maxW := range maxWorkers {
		mux := &sync.RWMutex{}
		completed := make(map[string]string)
		state := prepareState(mux, completed, maxW)

		state.AddTask("1", "test", "test", "user_1", "test", "test")
		state.AddTask("2", "test", "test", "user_1", "test", "test")
		state.AddTask("3", "test", "test", "user_2", "test", "test")
		state.AddTask("4", "test", "test", "user_3", "test", "test")
		state.AddTask("5", "test", "test", "user_4", "test", "test")
		state.AddTask("6", "test", "test", "user_5", "test", "test")
		state.AddTask("7", "test", "test", "user_5", "test", "test")
		state.AddTask("8", "test", "test", "user_5", "test", "test")
		state.AddTask("9", "test", "test", "user_5", "test", "test")
		state.AddTask("10", "test", "test", "user_5", "test", "test")

		for {
			if state.queuesEmpty() {
				break
			}
		}

		taskIDs := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
		for _, taskID := range taskIDs {
			_, ok := completed[taskID]

			if !ok {
				t.Errorf("Task %s was not completed!", taskID)
			}
		}
	}
}
