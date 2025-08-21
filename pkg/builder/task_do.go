package builder

import (
	"github.com/serverlessworkflow/sdk-go/v3/model"
)

func (d *DoTaskBuilder) Build() {}

type DoTaskBuilder struct {
	taskList *model.TaskList
}

func NewDoTaskBuilder(taskList *model.TaskList) *DoTaskBuilder {
	return &DoTaskBuilder{
		taskList: taskList,
	}
}
