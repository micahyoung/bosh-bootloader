package main_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry/bosh-bootloader/bbl/awsbackend"
	"github.com/rosenhouse/awsfaker"

	yaml "gopkg.in/yaml.v2"
)

var _ = Describe("bosh-deployment-vars", func() {
	var (
		fakeBOSH                 *fakeBOSHDirector
		fakeBOSHServer           *httptest.Server
		fakeBOSHCLIBackendServer *httptest.Server
	)

	BeforeEach(func() {
		fakeBOSH = &fakeBOSHDirector{}
		fakeBOSHServer = httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			fakeBOSH.ServeHTTP(responseWriter, request)
		}))

		fakeBOSHCLIBackendServer = httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		}))
	})

	Context("GCP", func() {
		BeforeEach(func() {
			args := []string{
				"--state-dir", tempDirectory,
				"--debug",
				"up",
				"--iaas", "gcp",
				"--gcp-service-account-key", serviceAccountKeyPath,
				"--gcp-project-id", "some-project-id",
				"--gcp-zone", "some-zone",
				"--gcp-region", "some-region",
			}
			executeCommand(args, 0)
		})

		It("prints a bosh create-env compatible vars-file", func() {
			args := []string{
				"--state-dir", tempDirectory,
				"bosh-deployment-vars",
			}
			executeCommand(args, 0)

			vars := gcp.BOSHDeploymentVars{}
			yaml.Unmarshal(stdout, vars)
			Expect(vars.InternalCIDR).To(Equal("some-internal-cidr"))
			Expect(vars.InternalGateway).To(Equal("some-internal-gateway"))
			Expect(vars.InternalIP).To(Equal("some-internal-ip"))
			Expect(vars.DirectorName).To(Equal("some-director-name"))
			Expect(vars.ExternalIP).To(Equal("some-external-ip"))
			Expect(vars.Zone).To(Equal("some-zone"))
			Expect(vars.Network).To(Equal("some-network"))
			Expect(vars.Subnetwork).To(Equal("some-subnetwork"))
			Expect(vars.Tags).To(Equal("some-tags"))
			Expect(vars.ProjectID).To(Equal("some-project-id"))
			Expect(vars.GCPCredentialsJSON).To(Equal("some-gcp-credentials-json"))
		})
	})

	Context("AWS", func() {
		var (
			fakeAWS       *awsbackend.Backend
			fakeAWSServer *httptest.Server
		)

		BeforeEach(func() {
			fakeAWS = awsbackend.New(fakeBOSHServer.URL)
			fakeAWSServer = httptest.NewServer(awsfaker.New(fakeAWS))

			upAWS(fakeAWSServer.URL, tempDirectory, 0)
		})

		It("prints a bosh create-env compatible vars-file", func() {
			args := []string{
				"--state-dir", tempDirectory,
				"bosh-deployment-vars",
			}
			executeCommand(args, 0)

			vars := aws.BOSHDeploymentVars{}
			yaml.Unmarshal(stdout, vars)
			Expect(vars.InternalCIDR).To(Equal("some-internal-cidr"))
			Expect(vars.InternalGateway).To(Equal("some-internal-gateway"))
			Expect(vars.InternalIP).To(Equal("some-internal-ip"))
			Expect(vars.DirectorName).To(Equal("some-director-name"))
			Expect(vars.ExternalIP).To(Equal("some-external-ip"))
			Expect(vars.AZ).To(Equal("some-az"))
			Expect(vars.SubnetID).To(Equal("some-subnet-id"))
			Expect(vars.AccessKeyID).To(Equal("some-access-key-id"))
			Expect(vars.SecretAccessKey).To(Equal("some-secret-access-key"))
			Expect(vars.DefaultKeyName).To(Equal("some-default-key-name"))
			Expect(vars.DefaultSecurityGroups).To(Equal("some-default-security-groups"))
			Expect(vars.Region).To(Equal("some-region"))
			Expect(vars.PrivateKey).To(Equal("some-private-key"))
		})
	})
})
