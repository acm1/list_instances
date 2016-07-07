package main

import (
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var regions = []string{
	"us-east-1",
	"us-west-2",
	"eu-west-1",
	"ap-northeast-1",
	"ap-northeast-2",
}

func perror(err error) {
	if err != nil {
		panic(err)
	}
}

func getInstances(region string, c chan *instance) {
	ec2Svc := ec2.New(
		session.New(
			request.WithRetryer(
				aws.NewConfig().WithRegion(region),
				client.DefaultRetryer{NumMaxRetries: 10},
			),
		),
	)

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

func main() {
	var is instances
	c := make(chan *instance)
	var wg sync.WaitGroup

	// Query EC2 endpoints in each region in parallel
	for _, r := range regions {
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
		is = append(is, i)
	}

	is.sort()
	is.printTable()
}
