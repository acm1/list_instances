package main

import (
	"net"

	"github.com/aws/aws-sdk-go/service/ec2"
)

type instance struct {
	*ec2.Instance
	name      string
	privateIP net.IP
}

func newInstance(i *ec2.Instance) (ret instance) {
	ret.Instance = i
	ret.privateIP = net.ParseIP(*i.PrivateIpAddress)
	for _, t := range i.Tags {
		if *t.Key == "Name" {
			ret.name = *t.Value
		}
	}
	return ret
}

func (i *instance) toRow() []string {
	return []string{
		i.name,
		*i.InstanceId,
		stringify(i.PublicIpAddress),
		*i.PrivateIpAddress,
		stringify(i.KeyName),
	}
}

func stringify(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// implement sort.Interface
type sortable []*instance

func (s sortable) Len() int      { return len(s) }
func (s sortable) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// Less sorts instances by name and then by private IP address
func (s sortable) Less(i, j int) bool {
	if s[i].name < s[j].name {
		return true
	}
	if s[i].name > s[j].name {
		return false
	}
	for n, v := range s[i].privateIP {
		if v < s[j].privateIP[n] {
			return true
		}
		if v > s[j].privateIP[n] {
			return false
		}
	}
	return false
}
