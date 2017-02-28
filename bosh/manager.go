package bosh

import (
	"errors"
	"fmt"

	yaml "gopkg.in/yaml.v2"

	"github.com/cloudfoundry/bosh-bootloader/aws/cloudformation"
	"github.com/cloudfoundry/bosh-bootloader/storage"
	"github.com/cloudfoundry/bosh-bootloader/terraform"
)

const (
	DIRECTOR_USERNAME = "admin"
)

type Manager struct {
	executor                executor
	terraformOutputProvider terraformOutputProvider
	stackManager            stackManager
}

type directorOutputs struct {
	directorPassword       string
	directorSSLCA          string
	directorSSLCertificate string
	directorSSLPrivateKey  string
}

type iaasInputs struct {
	InterpolateInput InterpolateInput
	DirectorAddress  string
}

type executor interface {
	Interpolate(InterpolateInput) (InterpolateOutput, error)
	CreateEnv(CreateEnvInput) (CreateEnvOutput, error)
	DeleteEnv(DeleteEnvInput) error
}

type terraformOutputProvider interface {
	Get(tfState, lbType string) (terraform.Outputs, error)
}

type stackManager interface {
	Describe(stackName string) (cloudformation.Stack, error)
}

func NewManager(executor executor, terraformOutputProvider terraformOutputProvider, stackManager stackManager) Manager {
	return Manager{
		executor:                executor,
		terraformOutputProvider: terraformOutputProvider,
		stackManager:            stackManager,
	}
}

func (m Manager) Create(state storage.State, opsFile []byte) (storage.State, error) {
	iaasInputs, err := m.generateIAASInputs(state)
	if err != nil {
		return storage.State{}, err
	}

	iaasInputs.InterpolateInput.OpsFile = opsFile

	interpolateOutputs, err := m.executor.Interpolate(iaasInputs.InterpolateInput)
	if err != nil {
		return storage.State{}, err
	}

	variables, err := yaml.Marshal(interpolateOutputs.Variables)
	createEnvOutputs, err := m.executor.CreateEnv(CreateEnvInput{
		Manifest:  interpolateOutputs.Manifest,
		State:     state.BOSH.State,
		Variables: string(variables),
	})
	switch err.(type) {
	case CreateEnvError:
		ceErr := err.(CreateEnvError)
		state.BOSH = storage.BOSH{
			Variables: string(variables),
			State:     ceErr.BOSHState(),
			Manifest:  interpolateOutputs.Manifest,
		}
		return storage.State{}, NewManagerCreateError(state, err)
	case error:
		return storage.State{}, err
	}

	directorOutputs := getDirectorOutputs(interpolateOutputs.Variables)

	state.BOSH = storage.BOSH{
		DirectorName:           fmt.Sprintf("bosh-%s", state.EnvID),
		DirectorAddress:        iaasInputs.DirectorAddress,
		DirectorUsername:       DIRECTOR_USERNAME,
		DirectorPassword:       directorOutputs.directorPassword,
		DirectorSSLCA:          directorOutputs.directorSSLCA,
		DirectorSSLCertificate: directorOutputs.directorSSLCertificate,
		DirectorSSLPrivateKey:  directorOutputs.directorSSLPrivateKey,
		Variables:              string(variables),
		State:                  createEnvOutputs.State,
		Manifest:               interpolateOutputs.Manifest,
	}

	return state, nil
}

func (m Manager) Delete(state storage.State) error {
	err := m.executor.DeleteEnv(DeleteEnvInput{
		Manifest:  state.BOSH.Manifest,
		State:     state.BOSH.State,
		Variables: state.BOSH.Variables,
	})
	switch err.(type) {
	case DeleteEnvError:
		deErr := err.(DeleteEnvError)
		state.BOSH.State = deErr.BOSHState()
		return NewManagerDeleteError(state, err)
	case error:
		return err
	}

	return nil
}

func (m Manager) generateIAASInputs(state storage.State) (iaasInputs, error) {
	switch state.IAAS {
	case "gcp":
		terraformOutputs, err := m.terraformOutputProvider.Get(state.TFState, state.LB.Type)
		if err != nil {
			return iaasInputs{}, err
		}
		return iaasInputs{
			InterpolateInput: InterpolateInput{
				IAAS:         state.IAAS,
				DirectorName: fmt.Sprintf("bosh-%s", state.EnvID),
				Zone:         state.GCP.Zone,
				Network:      terraformOutputs.NetworkName,
				Subnetwork:   terraformOutputs.SubnetworkName,
				Tags: []string{
					terraformOutputs.BOSHTag,
					terraformOutputs.InternalTag,
				},
				ProjectID:       state.GCP.ProjectID,
				ExternalIP:      terraformOutputs.ExternalIP,
				CredentialsJSON: state.GCP.ServiceAccountKey,
				PrivateKey:      state.KeyPair.PrivateKey,
				BOSHState:       state.BOSH.State,
				Variables:       state.BOSH.Variables,
			},
			DirectorAddress: terraformOutputs.DirectorAddress,
		}, nil
	case "aws":
		stack, err := m.stackManager.Describe(state.Stack.Name)
		if err != nil {
			return iaasInputs{}, err
		}
		return iaasInputs{
			InterpolateInput: InterpolateInput{
				IAAS:                  state.IAAS,
				DirectorName:          fmt.Sprintf("bosh-%s", state.EnvID),
				AZ:                    stack.Outputs["BOSHSubnetAZ"],
				AccessKeyID:           stack.Outputs["BOSHUserAccessKey"],
				SecretAccessKey:       stack.Outputs["BOSHUserSecretAccessKey"],
				Region:                state.AWS.Region,
				DefaultKeyName:        state.KeyPair.Name,
				DefaultSecurityGroups: []string{stack.Outputs["BOSHSecurityGroup"]},
				SubnetID:              stack.Outputs["BOSHSubnet"],
				ExternalIP:            stack.Outputs["BOSHEIP"],
				PrivateKey:            state.KeyPair.PrivateKey,
				BOSHState:             state.BOSH.State,
				Variables:             state.BOSH.Variables,
			},
			DirectorAddress: stack.Outputs["BOSHURL"],
		}, nil
	default:
		return iaasInputs{}, errors.New("A valid IAAS was not provided")
	}
}

func getDirectorOutputs(variables map[interface{}]interface{}) directorOutputs {
	directorSSLInterfaceMap := variables["director_ssl"].(map[interface{}]interface{})
	directorSSL := map[string]string{}
	for k, v := range directorSSLInterfaceMap {
		directorSSL[k.(string)] = v.(string)
	}

	return directorOutputs{
		directorPassword:       variables["admin_password"].(string),
		directorSSLCA:          directorSSL["ca"],
		directorSSLCertificate: directorSSL["certificate"],
		directorSSLPrivateKey:  directorSSL["private_key"],
	}
}