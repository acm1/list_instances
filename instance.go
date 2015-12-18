package main

import (
	"strconv"
	"strings"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type instance struct {
	*ec2.Instance
	name *string
}

type instanceSlice []*instance

func newInstance(i *ec2.Instance) (ret *instance) {
	ret = new(instance)
	ret.Instance = i
	for _, t := range i.Tags {
		if *t.Key == "Name" {
			ret.name = t.Value
		}
	}
	return
}

func (i *instance) toRow() []string {
	return []string{
		*i.name,
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

func (i *instance) privateIPOctets() []int {
	stringOctets := strings.Split(*i.PrivateIpAddress, ".")
	intOctets := make([]int, 4)
	for n, o := range stringOctets {
		x, _ := strconv.ParseInt(o, 10, 32)
		intOctets[n] = int(x)
	}
	return intOctets
}

func (s instanceSlice) Len() int {
	return len(s)
}

func (s instanceSlice) Less(i, j int) bool {
	octetsI, octetsJ := s[i].privateIPOctets(), s[j].privateIPOctets()
	if *s[i].name < *s[j].name {
		return true
	}
	if *s[i].name > *s[j].name {
		return false
	}
	for n := 0; n < 3; n++ {
		if octetsI[n] < octetsJ[n] {
			return true
		}
		if octetsI[n] > octetsJ[n] {
			return false
		}
	}
	return true
}

func (s instanceSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
