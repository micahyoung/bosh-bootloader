package main_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("bbl latest-error", func() {
	var (
		tempDirectory         string
		serviceAccountKeyPath string
		fakeBOSHServer        *httptest.Server
		fakeBOSH              *fakeBOSHDirector

		fastFail                 bool
		fastFailMutex            sync.Mutex
		callRealInterpolate      bool
		callRealInterpolateMutex sync.Mutex

		createEnvArgs   string
		interpolateArgs []string
	)

	BeforeEach(func() {
		var err error

		fakeBOSH = &fakeBOSHDirector{}
		fakeBOSHServer = httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			fakeBOSH.ServeHTTP(responseWriter, request)
		}))

		fakeBOSHCLIBackendServer.SetHandler(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			switch request.URL.Path {
			case "/version":
				responseWriter.Write([]byte("v2.0.0"))
			case "/path":
				responseWriter.Write([]byte(noFakesPath))
			case "/create-env/args":
				body, err := ioutil.ReadAll(request.Body)
				Expect(err).NotTo(HaveOccurred())
				createEnvArgs = string(body)
			case "/interpolate/args":
				body, err := ioutil.ReadAll(request.Body)
				Expect(err).NotTo(HaveOccurred())
				interpolateArgs = append(interpolateArgs, string(body))
			case "/create-env/fastfail":
				fastFailMutex.Lock()
				defer fastFailMutex.Unlock()
				if fastFail {
					responseWriter.WriteHeader(http.StatusInternalServerError)
				} else {
					responseWriter.WriteHeader(http.StatusOK)
				}
				return
			case "/call-real-interpolate":
				callRealInterpolateMutex.Lock()
				defer callRealInterpolateMutex.Unlock()
				if callRealInterpolate {
					responseWriter.Write([]byte("true"))
				} else {
					responseWriter.Write([]byte("false"))
				}
			}
		}))

		fakeTerraformBackendServer.SetHandler(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			switch request.URL.Path {
			case "/output/external_ip":
				responseWriter.Write([]byte("127.0.0.1"))
			case "/output/director_address":
				responseWriter.Write([]byte(fakeBOSHServer.URL))
			case "/output/network_name":
				responseWriter.Write([]byte("some-network-name"))
			case "/output/subnetwork_name":
				responseWriter.Write([]byte("some-subnetwork-name"))
			case "/output/internal_tag_name":
				responseWriter.Write([]byte("some-internal-tag"))
			case "/output/bosh_open_tag_name":
				responseWriter.Write([]byte("some-bosh-tag"))
			case "/version":
				responseWriter.Write([]byte("0.8.6"))
			}
		}))

		tempDirectory, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())

		tempFile, err := ioutil.TempFile("", "gcpServiceAccountKey")
		Expect(err).NotTo(HaveOccurred())

		serviceAccountKeyPath = tempFile.Name()
		err = ioutil.WriteFile(serviceAccountKeyPath, []byte(serviceAccountKey), os.ModePerm)
		Expect(err).NotTo(HaveOccurred())

		fastFailMutex.Lock()
		defer fastFailMutex.Unlock()
		fastFail = false

		args := []string{
			"--state-dir", tempDirectory,
			"--debug",
			"up",
			"--iaas", "gcp",
			"--gcp-service-account-key", serviceAccountKeyPath,
			"--gcp-project-id", "some-project-id",
			"--gcp-zone", "some-zone",
			"--gcp-region", "us-west1",
		}

		executeCommand(args, 0)
	})

	It("prints the terraform output from the last command", func() {
		args := []string{
			"--state-dir", tempDirectory,
			"latest-error",
		}

		session := executeCommand(args, 0)

		Expect(session.Out.Contents()).To(ContainSubstring("some terraform output"))
		Expect(session.Err.Contents()).To(ContainSubstring("some terraform error output"))
	})
})
