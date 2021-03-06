package terraform

import (
	"errors"

	"github.com/cloudfoundry/bosh-bootloader/storage"
	"github.com/coreos/go-semver/semver"
)

type Manager struct {
	executor          executor
	templateGenerator templateGenerator
	inputGenerator    inputGenerator
	outputGenerator   outputGenerator
	logger            logger
}

type executor interface {
	Version() (string, error)
	Destroy(inputs map[string]string, terraformTemplate, tfState string) (string, error)
	Apply(inputs map[string]string, terraformTemplate, tfState string) (string, error)
}

type templateGenerator interface {
	Generate(storage.State) string
}

type inputGenerator interface {
	Generate(storage.State) (map[string]string, error)
}

type outputGenerator interface {
	Generate(storage.State) (map[string]interface{}, error)
}

type logger interface {
	Step(string, ...interface{})
}

func NewManager(executor executor, templateGenerator templateGenerator, inputGenerator inputGenerator, outputGenerator outputGenerator, logger logger) Manager {
	return Manager{
		executor:          executor,
		templateGenerator: templateGenerator,
		inputGenerator:    inputGenerator,
		outputGenerator:   outputGenerator,
		logger:            logger,
	}
}

func (m Manager) Version() (string, error) {
	return m.executor.Version()
}

func (m Manager) ValidateVersion() error {
	version, err := m.executor.Version()
	if err != nil {
		return err
	}

	currentVersion, err := semver.NewVersion(version)
	if err != nil {
		return err
	}

	// This shouldn't fail, so there is no test for capturing the error.
	minimumVersion, err := semver.NewVersion("0.8.5")
	if err != nil {
		return err
	}

	if currentVersion.LessThan(*minimumVersion) {
		return errors.New("Terraform version must be at least v0.8.5")
	}

	// This shouldn't fail, so there is no test for capturing the error.
	blacklistedVersion, err := semver.NewVersion("0.9.0")
	if err != nil {
		return err
	}

	if currentVersion.Equal(*blacklistedVersion) {
		return errors.New("Version 0.9.0 of terraform is incompatible with bbl, please try a later version.")
	}

	return nil
}

func (m Manager) Apply(bblState storage.State) (storage.State, error) {
	m.logger.Step("generating terraform template")
	template := m.templateGenerator.Generate(bblState)

	input, err := m.inputGenerator.Generate(bblState)
	if err != nil {
		return storage.State{}, err
	}

	tfState, err := m.executor.Apply(
		input,
		template,
		bblState.TFState)

	switch err.(type) {
	case executorError:
		return storage.State{}, NewManagerError(bblState, err.(executorError))
	case error:
		return storage.State{}, err
	}
	m.logger.Step("applied terraform template")

	bblState.TFState = tfState
	return bblState, nil
}

func (m Manager) Destroy(bblState storage.State) (storage.State, error) {
	m.logger.Step("destroying infrastructure")
	if bblState.TFState == "" {
		return bblState, nil
	}

	template := m.templateGenerator.Generate(bblState)

	input, err := m.inputGenerator.Generate(bblState)
	if err != nil {
		return storage.State{}, err
	}

	tfState, err := m.executor.Destroy(
		input,
		template,
		bblState.TFState)
	switch err.(type) {
	case executorError:
		return storage.State{}, NewManagerError(bblState, err.(executorError))
	case error:
		return storage.State{}, err
	}
	m.logger.Step("finished destroying infrastructure")

	bblState.TFState = tfState
	return bblState, nil
}

func (m Manager) GetOutputs(bblState storage.State) (map[string]interface{}, error) {
	outputs, err := m.outputGenerator.Generate(bblState)
	if err != nil {
		return map[string]interface{}{}, err
	}

	return outputs, nil
}
