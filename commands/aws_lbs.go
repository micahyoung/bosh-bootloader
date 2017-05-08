package commands

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cloudfoundry/bosh-bootloader/storage"
)

type AWSLBs struct {
	credentialValidator   credentialValidator
	infrastructureManager infrastructureManager
	terraformManager      terraformManager
	logger                logger
}

func NewAWSLBs(credentialValidator credentialValidator, infrastructureManager infrastructureManager, terraformManager terraformManager, logger logger) AWSLBs {
	return AWSLBs{
		credentialValidator:   credentialValidator,
		infrastructureManager: infrastructureManager,
		terraformManager:      terraformManager,
		logger:                logger,
	}
}

func (l AWSLBs) Execute(subcommandFlags []string, state storage.State) error {
	err := l.credentialValidator.Validate()
	if err != nil {
		return err
	}

	if state.TFState != "" {
		terraformOutputs, err := l.terraformManager.GetOutputs(state)
		if err != nil {
			return err
		}

		switch state.LB.Type {
		case "cf":
			if len(subcommandFlags) > 0 && subcommandFlags[0] == "--json" {
				lbOutput, err := json.Marshal(struct {
					RouterLBName           string   `json:"cf_router_load_balancer,omitempty"`
					RouterLBURL            string   `json:"cf_router_load_balancer_url,omitempty"`
					SSHProxyLBName         string   `json:"cf_ssh_proxy_load_balancer,omitempty"`
					SSHProxyLBURL          string   `json:"cf_ssh_proxy_load_balancer_url,omitempty"`
					SystemDomainDNSServers []string `json:"system_domain_dns_servers,omitempty"`
				}{
					RouterLBName:           terraformOutputs["cf_router_lb_name"].(string),
					RouterLBURL:            terraformOutputs["cf_router_lb_url"].(string),
					SSHProxyLBName:         terraformOutputs["cf_ssh_lb_name"].(string),
					SSHProxyLBURL:          terraformOutputs["cf_ssh_lb_url"].(string),
					SystemDomainDNSServers: terraformOutputs["env_dns_zone_name_servers"].([]string),
				})
				if err != nil {
					// not tested
					return err
				}

				l.logger.Println(string(lbOutput))
			} else {
				fmt.Printf("=======================%s", terraformOutputs)
				l.logger.Printf("CF Router LB Name: %s\n", terraformOutputs["cf_router_load_balancer"])
				l.logger.Printf("CF Router LB URL: %s\n", terraformOutputs["cf_router_load_balancer_url"])
				l.logger.Printf("CF SSH Proxy LB Name: %s\n", terraformOutputs["cf_ssh_proxy_load_balancer"])
				l.logger.Printf("CF SSH Proxy LB URL: %s\n", terraformOutputs["cf_ssh_proxy_load_balancer_url"])

				servers := terraformOutputs["system_domain_dns_servers"]
				for _, s := range servers {
					fmt.Printf("&&&&&&&&&&&&&&&&&&&&&&&%s", s)
				}
				if dnsServers, ok := terraformOutputs["system_domain_dns_servers"]; ok {
					l.logger.Printf("CF System Domain DNS servers: %s\n", dnsServers)
				}
			}
		case "concourse":
			l.logger.Printf("Concourse LB Name: %s\n", terraformOutputs["concourse_load_balancer"])
			l.logger.Printf("Concourse LB URL: %s\n", terraformOutputs["concourse_load_balancer_url"])
		default:
			return errors.New("no lbs found")
		}

	} else {
		stack, err := l.infrastructureManager.Describe(state.Stack.Name)
		if err != nil {
			return err
		}

		switch state.Stack.LBType {
		case "cf":
			l.logger.Printf("CF Router LB: %s [%s]\n", stack.Outputs["CFRouterLoadBalancer"], stack.Outputs["CFRouterLoadBalancerURL"])
			l.logger.Printf("CF SSH Proxy LB: %s [%s]\n", stack.Outputs["CFSSHProxyLoadBalancer"], stack.Outputs["CFSSHProxyLoadBalancerURL"])
		case "concourse":
			l.logger.Printf("Concourse LB: %s [%s]\n", stack.Outputs["ConcourseLoadBalancer"], stack.Outputs["ConcourseLoadBalancerURL"])
		default:
			return errors.New("no lbs found")
		}
	}
	return nil
}
