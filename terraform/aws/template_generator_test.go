package aws_test

import (
	"io/ioutil"

	"github.com/cloudfoundry/bosh-bootloader/storage"
	"github.com/cloudfoundry/bosh-bootloader/terraform/aws"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("TemplateGenerator", func() {
	var (
		templateGenerator aws.TemplateGenerator
	)

	BeforeEach(func() {
		templateGenerator = aws.NewTemplateGenerator()
	})

	Describe("Generate", func() {
		DescribeTable("generates a terraform template for aws",
			func(fixtureFilename, lbType string) {
				expectedTemplate, err := ioutil.ReadFile(fixtureFilename)
				Expect(err).NotTo(HaveOccurred())

				template := templateGenerator.Generate(storage.State{
					LB: storage.LB{
						Type: lbType,
					},
				})
				Expect(template).To(Equal(string(expectedTemplate)))
			},
			Entry("when no lb type is provided", "fixtures/template_no_lb.tf", ""),
			Entry("when a concourse lb type is provided", "fixtures/template_concourse_lb.tf", "concourse"),
			Entry("when a cf lb type is provided", "fixtures/template_cf_lb.tf", "cf"),
		)
	})
})
