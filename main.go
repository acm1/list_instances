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

func getInstances(region string, c chan *instance) {
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
			c <- &inst
		}
	}
}

func printTable(s []*instance) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Id", "PublicIP", "PrivateIP", "Key"})
	for _, i := range s {
		table.Append(i.toRow())
	}
	table.Render()
}

func main() {
	var instances []*instance
	c := make(chan *instance)
	var wg sync.WaitGroup

	// Query EC2 endpoints in each region in parallel
	for _, r := range REGIONS {
		wg.Add(1)
		go func(r string) {
			getInstances(r, c)
			wg.Done()
		}(r)
	}
	// Close channel after all EC2 queries have returned
	go func() {
		wg.Wait()
		close(c)
	}()
	// Build up a slice of instances as they are returned to us
	for {
		i, ok := <-c
		if !ok {
			break
		}
		instances = append(instances, i)
	}

	sort.Sort(sortable(instances))
	printTable(instances)
}
