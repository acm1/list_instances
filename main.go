package main

import (
	"os"
	"sort"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/olekukonko/tablewriter"
)

var REGIONS []string = []string{"us-east-1", "eu-west-1", "ap-northeast-1"}

func perror(err error) {
	if err != nil {
		panic(err)
	}
}

func getInstances(region string, instanceChan chan *instance) {
	ec2Svc := ec2.New(&aws.Config{Region: &region})

	ec2instances, err := ec2Svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String("running"), aws.String("pending")},
			},
		},
	})
	perror(err)

	for _, r := range ec2instances.Reservations {
		for _, i := range r.Instances {
			inst := newInstance(i)
			instanceChan <- inst
		}
	}
}

func main() {
	var instances instanceSlice
	instanceChan := make(chan *instance)
	var wg sync.WaitGroup

	for _, r := range REGIONS {
		wg.Add(1)
		go func(r string) {
			getInstances(r, instanceChan)
			wg.Done()
		}(r)
	}
	go func() {
		for {
			i, ok := <-instanceChan
			if !ok {
				return
			}
			instances = append(instances, i)
		}
	}()
	wg.Wait()
	close(instanceChan)

	sort.Sort(instances)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Id", "PublicIP", "PrivateIP", "Key"})
	for _, i := range instances {
		table.Append(i.toRow())
	}
	table.Render()
}
