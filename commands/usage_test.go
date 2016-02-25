package commands_test

import (
	"bytes"
	"strings"

	"github.com/pivotal-cf-experimental/bosh-bootloader/commands"
	"github.com/pivotal-cf-experimental/bosh-bootloader/state"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Usage", func() {
	var (
		usage  commands.Usage
		stdout *bytes.Buffer
	)

	BeforeEach(func() {
		stdout = bytes.NewBuffer([]byte{})
		usage = commands.NewUsage(stdout)
	})

	Describe("Execute", func() {
		It("prints out the usage information", func() {
			_, err := usage.Execute(commands.GlobalFlags{}, state.State{})
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout.String()).To(Equal(strings.TrimSpace(`
Usage:
  bbl [GLOBAL OPTIONS] COMMAND [OPTIONS]

Global Options:
  --help    [-h] "print usage"
  --version [-v] "print version"

  --aws-access-key-id     "AWS AccessKeyID value"
  --aws-secret-access-key "AWS SecretAccessKey value"
  --aws-region            "AWS Region"
  --state-dir             "Directory that stores the state.json"

Commands:
  help                                     "print usage"
  version                                  "print version"
  unsupported-print-concourse-aws-template "print a concourse aws template"
  unsupported-create-bosh-aws-keypair      "create and upload a keypair to AWS"
  unsupported-provision-aws-for-concourse  "create a new concourse stack on AWS"
`)))
		})

		It("returns the given state unmodified", func() {
			s, err := usage.Execute(commands.GlobalFlags{}, state.State{
				Version: 2,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(s).To(Equal(state.State{
				Version: 2,
			}))
		})
	})
})
