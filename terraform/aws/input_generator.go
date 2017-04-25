package aws

import (
	"encoding/json"

	"github.com/cloudfoundry/bosh-bootloader/storage"
)

type InputGenerator struct {
	availabilityZoneRetriever availabilityZoneRetriever
}

type availabilityZoneRetriever interface {
	Retrieve(string) ([]string, error)
}

var jsonMarshal = json.Marshal

func NewInputGenerator(availabilityZoneRetriever availabilityZoneRetriever) InputGenerator {
	return InputGenerator{
		availabilityZoneRetriever: availabilityZoneRetriever,
	}
}

func (i InputGenerator) Generate(state storage.State) (map[string]string, error) {
	azs, err := i.availabilityZoneRetriever.Retrieve(state.AWS.Region)
	if err != nil {
		return map[string]string{}, err
	}

	azsString, err := jsonMarshal(azs)
	if err != nil {
		return map[string]string{}, err
	}

	inputs := map[string]string{
		"env_id":                 state.EnvID,
		"nat_ssh_key_pair_name":  state.KeyPair.Name,
		"access_key":             state.AWS.AccessKeyID,
		"secret_key":             state.AWS.SecretAccessKey,
		"region":                 state.AWS.Region,
		"bosh_availability_zone": state.Stack.BOSHAZ,
		"availability_zones":     string(azsString),
	}

	if state.LB.Type == "cf" || state.LB.Type == "concourse" {
		inputs["ssl_certificate"] = state.LB.Cert
		inputs["ssl_certificate_private_key"] = state.LB.Key
	}

	if state.LB.Type == "cf" {
		inputs["ssl_certificate_chain"] = state.LB.Chain
		if state.LB.Domain != "" {
			inputs["system_domain"] = state.LB.Domain
		}
	}

	return inputs, nil
}
