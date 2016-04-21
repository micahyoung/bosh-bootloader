package templates

import "fmt"

type LoadBalancerTemplateBuilder struct{}

func NewLoadBalancerTemplateBuilder() LoadBalancerTemplateBuilder {
	return LoadBalancerTemplateBuilder{}
}

func (LoadBalancerTemplateBuilder) CFSSHProxyLoadBalancer(numberOfAvailabilityZones int) Template {
	subnets := []interface{}{}
	for i := 1; i <= numberOfAvailabilityZones; i++ {
		subnets = append(subnets, Ref{fmt.Sprintf("LoadBalancerSubnet%d", i)})
	}

	return Template{
		Outputs: map[string]Output{
			"CFSSHProxyLoadBalancer": {Value: Ref{"CFSSHProxyLoadBalancer"}},
			"CFSSHProxyLoadBalancerURL": {
				Value: FnGetAtt{
					[]string{
						"CFSSHProxyLoadBalancer",
						"DNSName",
					},
				},
			},
		},
		Resources: map[string]Resource{
			"CFSSHProxyLoadBalancer": {
				Type: "AWS::ElasticLoadBalancing::LoadBalancer",
				Properties: ElasticLoadBalancingLoadBalancer{
					CrossZone:      true,
					Subnets:        subnets,
					SecurityGroups: []interface{}{Ref{"CFSSHProxySecurityGroup"}},

					HealthCheck: HealthCheck{
						HealthyThreshold:   "5",
						Interval:           "6",
						Target:             "tcp:2222",
						Timeout:            "2",
						UnhealthyThreshold: "2",
					},

					Listeners: []Listener{
						{
							Protocol:         "tcp",
							LoadBalancerPort: "2222",
							InstanceProtocol: "tcp",
							InstancePort:     "2222",
						},
					},
				},
			},
		},
	}
}

func (LoadBalancerTemplateBuilder) CFRouterLoadBalancer(numberOfAvailabilityZones int) Template {
	subnets := []interface{}{}
	for i := 1; i <= numberOfAvailabilityZones; i++ {
		subnets = append(subnets, Ref{fmt.Sprintf("LoadBalancerSubnet%d", i)})
	}

	return Template{
		Outputs: map[string]Output{
			"CFRouterLoadBalancer": {Value: Ref{"CFRouterLoadBalancer"}},
			"CFRouterLoadBalancerURL": {
				Value: FnGetAtt{
					[]string{
						"CFRouterLoadBalancer",
						"DNSName",
					},
				},
			},
		},
		Resources: map[string]Resource{
			"CFRouterLoadBalancer": {
				Type: "AWS::ElasticLoadBalancing::LoadBalancer",
				Properties: ElasticLoadBalancingLoadBalancer{
					CrossZone:      true,
					Subnets:        subnets,
					SecurityGroups: []interface{}{Ref{"CFRouterSecurityGroup"}},

					HealthCheck: HealthCheck{
						HealthyThreshold:   "5",
						Interval:           "12",
						Target:             "tcp:80",
						Timeout:            "2",
						UnhealthyThreshold: "2",
					},

					Listeners: []Listener{
						{
							Protocol:         "http",
							LoadBalancerPort: "80",
							InstanceProtocol: "http",
							InstancePort:     "80",
						},
					},
				},
			},
		},
	}
}

func (LoadBalancerTemplateBuilder) ConcourseLoadBalancer(numberOfAvailabliltyZones int) Template {
	subnets := []interface{}{}
	for i := 1; i <= numberOfAvailabliltyZones; i++ {
		subnets = append(subnets, Ref{fmt.Sprintf("LoadBalancerSubnet%d", i)})
	}

	return Template{
		Outputs: map[string]Output{
			"ConcourseLoadBalancer": {Value: Ref{"ConcourseLoadBalancer"}},
			"ConcourseLoadBalancerURL": {
				Value: FnGetAtt{
					[]string{
						"ConcourseLoadBalancer",
						"DNSName",
					},
				},
			},
		},
		Resources: map[string]Resource{
			"ConcourseLoadBalancer": {
				Type: "AWS::ElasticLoadBalancing::LoadBalancer",
				Properties: ElasticLoadBalancingLoadBalancer{
					Subnets:        subnets,
					SecurityGroups: []interface{}{Ref{"ConcourseSecurityGroup"}},

					HealthCheck: HealthCheck{
						HealthyThreshold:   "2",
						Interval:           "30",
						Target:             "tcp:8080",
						Timeout:            "5",
						UnhealthyThreshold: "10",
					},

					Listeners: []Listener{
						{
							Protocol:         "tcp",
							LoadBalancerPort: "80",
							InstanceProtocol: "tcp",
							InstancePort:     "8080",
						},
						{
							Protocol:         "tcp",
							LoadBalancerPort: "2222",
							InstanceProtocol: "tcp",
							InstancePort:     "2222",
						},
					},
				},
			},
		},
	}
}