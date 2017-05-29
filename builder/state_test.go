package builder

import (
	"github.com/Sirupsen/logrus"
	"testing"
	"time"
)

func prepareState(maxWorkers int) *State {
	processor := func(logger logrus.FieldLogger, task Task) {
		time.Sleep(3 * time.Second)
	}
	state := NewState(processor, logrus.WithField("test", "test"), maxWorkers)
	return state
}

func TestState(t *testing.T) {
	//prepareState().AddTask()
}

func TestAddTask(t *testing.T) {
	state := prepareState(2)

	state.AddTask("test", "test", "test", "test", "test")
	if len(state.tasks) != 1 {
		t.Fail()
	}

	state.AddTask("test", "test", "test", "test", "test")

	for {
		go state.ProcessPool()
	}

	state.Shutdown()
	if len(state.tasks) != 0 {
		t.Fail()
	}
	if len(state.progress) != 0 {
		t.Fail()
	}
}
