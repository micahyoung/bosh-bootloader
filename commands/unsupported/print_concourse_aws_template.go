package unsupported

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/pivotal-cf-experimental/bosh-bootloader/aws/cloudformation"
	"github.com/pivotal-cf-experimental/bosh-bootloader/commands"
	"github.com/pivotal-cf-experimental/bosh-bootloader/state"
)

type templateBuilder interface {
	Build() cloudformation.Template
}

type PrintConcourseAWSTemplate struct {
	stdout  io.Writer
	builder templateBuilder
}

func NewPrintConcourseAWSTemplate(stdout io.Writer, builder templateBuilder) PrintConcourseAWSTemplate {
	return PrintConcourseAWSTemplate{
		stdout:  stdout,
		builder: builder,
	}
}

func (c PrintConcourseAWSTemplate) Execute(globalFlags commands.GlobalFlags, s state.State) (state.State, error) {
	template := c.builder.Build()
	buf, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return state.State{}, err
	}

	fmt.Fprintf(c.stdout, string(buf))
	return s, nil
}
