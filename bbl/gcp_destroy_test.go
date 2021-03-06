package main_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"

	"github.com/cloudfoundry/bosh-bootloader/storage"
	"github.com/cloudfoundry/bosh-bootloader/testhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("bbl destroy gcp", func() {
	var (
		state          storage.State
		tempDirectory  string
		statePath      string
		fakeBOSHServer *httptest.Server
		fakeBOSH       *fakeBOSHDirector
		fastFail       bool
		fastFailMutex  sync.Mutex
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
				responseWriter.Write([]byte("2.0.0"))
			case "/delete-env/fastfail":
				fastFailMutex.Lock()
				defer fastFailMutex.Unlock()
				if fastFail {
					responseWriter.WriteHeader(http.StatusInternalServerError)
				} else {
					responseWriter.WriteHeader(http.StatusOK)
				}
				return
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

		variables := `
admin_password: rhkj9ys4l9guqfpc9vmp
director_ssl:
  certificate: some-certificate
  private_key: some-private-key
  ca: some-ca
`

		state = storage.State{
			Version: 3,
			IAAS:    "gcp",
			TFState: `{"key": "value"}`,
			GCP: storage.GCP{
				ProjectID:         "some-project-id",
				ServiceAccountKey: serviceAccountKey,
				Region:            "some-region",
				Zone:              "some-zone",
			},
			KeyPair: storage.KeyPair{
				Name:       "some-keypair-name",
				PrivateKey: testhelpers.BBL_KEY,
			},
			BOSH: storage.BOSH{
				DirectorName: "some-bosh-director-name",
				Variables:    variables,
				State: map[string]interface{}{
					"new-key": "new-value",
				},
			},
		}

		stateContents, err := json.Marshal(state)
		Expect(err).NotTo(HaveOccurred())

		statePath = filepath.Join(tempDirectory, "bbl-state.json")
		err = ioutil.WriteFile(statePath, stateContents, os.ModePerm)
		Expect(err).NotTo(HaveOccurred())
	})

	It("deletes the bbl-state", func() {
		args := []string{
			"--state-dir", tempDirectory,
			"destroy", "--no-confirm",
		}
		_ = executeCommand(args, 0)

		_, err := os.Stat(statePath)
		Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
	})

	It("calls out to terraform", func() {
		args := []string{
			"--state-dir", tempDirectory,
			"--debug",
			"destroy", "--no-confirm",
		}
		session := executeCommand(args, 0)

		Expect(session.Out.Contents()).To(ContainSubstring("terraform destroy"))
	})

	Context("when the bbl-state does not contain a TFState", func() {
		var (
			args []string
		)

		BeforeEach(func() {
			state.TFState = ""
			stateContents, err := json.Marshal(state)
			Expect(err).NotTo(HaveOccurred())

			statePath = filepath.Join(tempDirectory, "bbl-state.json")
			err = ioutil.WriteFile(statePath, stateContents, os.ModePerm)
			Expect(err).NotTo(HaveOccurred())
			args = []string{
				"--debug",
				"--state-dir", tempDirectory,
				"destroy", "--no-confirm",
			}
		})

		It("succeeds", func() {
			executeCommand(args, 0)
		})
	})

	Context("bbl re-entrance", func() {
		Context("when terraform fails", func() {
			var (
				args    []string
				session *gexec.Session
			)

			BeforeEach(func() {
				state.GCP.Region = "fail-to-terraform"

				stateContents, err := json.Marshal(state)
				Expect(err).NotTo(HaveOccurred())

				statePath = filepath.Join(tempDirectory, "bbl-state.json")
				err = ioutil.WriteFile(statePath, stateContents, os.ModePerm)
				Expect(err).NotTo(HaveOccurred())
				args = []string{
					"--state-dir", tempDirectory,
					"destroy", "--no-confirm",
				}

				session = executeCommand(args, 1)
			})

			It("saves the tf state when terraform destroy fails with ManagerError", func() {
				state = readStateJson(tempDirectory)
				Expect(state.TFState).To(Equal(`{"key":"partial-apply"}`))
			})

			Context("when no --debug is provided", func() {
				It("returns a helpful error message", func() {
					Expect(session.Err.Contents()).To(ContainSubstring("use --debug for additional debug output"))
				})
			})
		})

		Context("when bosh fails", func() {
			BeforeEach(func() {
				fastFailMutex.Lock()
				fastFail = true
				fastFailMutex.Unlock()

				args := []string{
					"--debug",
					"--state-dir", tempDirectory,
					"destroy", "--no-confirm",
				}

				executeCommand(args, 1)
			})

			AfterEach(func() {
				fastFailMutex.Lock()
				fastFail = false
				fastFailMutex.Unlock()
			})

			It("stores a partial bosh state", func() {
				state := readStateJson(tempDirectory)
				Expect(state.BOSH.State).To(Equal(map[string]interface{}{
					"partial": "bosh-state",
				}))
			})
		})
	})

	Context("when the bosh cli version is <2.0", func() {
		BeforeEach(func() {
			fakeBOSHCLIBackendServer.SetHandler(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
				switch request.URL.Path {
				case "/version":
					responseWriter.Write([]byte("1.9.0"))
				}
			}))
		})

		It("fast fails with a helpful error message", func() {
			args := []string{
				"--state-dir", tempDirectory,
				"destroy", "--no-confirm",
			}

			session := executeCommand(args, 1)

			Expect(session.Err.Contents()).To(ContainSubstring("BOSH version must be at least v2.0.0"))
		})
	})

	Context("when the terraform version is <0.8.5", func() {
		BeforeEach(func() {
			fakeTerraformBackendServer.SetHandler(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
				switch request.URL.Path {
				case "/version":
					responseWriter.Write([]byte("0.8.4"))
				}
			}))
		})

		It("fast fails with a helpful error message", func() {
			args := []string{
				"--state-dir", tempDirectory,
				"destroy", "--no-confirm",
			}

			session := executeCommand(args, 1)

			Expect(session.Err.Contents()).To(ContainSubstring("Terraform version must be at least v0.8.5"))
		})
	})

	Context("when instances exist in my bbl environment", func() {
		BeforeEach(func() {
			gcpBackend.HandleListInstances(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"items": [
						{
							"name": "some-vm",
							"networkInterfaces": [
								{
								 "network": "https://www.googleapis.com/compute/v1/projects/some-project-id/global/networks/some-network-name"
								}
							],
							"metadata": {
								"items": [
									{
										"key": "director",
										"value": "some-director"
									}
								]
							}
						}
					]
				}`))
			})
		})

		AfterEach(func() {
			gcpBackend.HandleListInstances(nil)
		})

		It("fast fails with a nice error message", func() {
			args := []string{
				"--state-dir", tempDirectory,
				"destroy", "--no-confirm",
			}
			session := executeCommand(args, 1)

			Expect(session.Err.Contents()).To(ContainSubstring("bbl environment is not safe to delete; vms still exist in network"))
		})
	})
})
