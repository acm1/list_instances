package main

import (
	"sync"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"

	log "github.com/Sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var DEFAULT_REGIONS = []string{
	"us-east-1",
	"us-east-2",
	"us-west-2",
	"eu-west-1",
	"ap-northeast-1",
	"ap-northeast-2",
}

var (
	debug   = kingpin.Flag("debug", "Enable debug mode.").Bool()
	regions = kingpin.Flag("region", "AWS Region to query. Repeat for multiple regions.").Short('r').Default(DEFAULT_REGIONS...).Strings()
	retries = kingpin.Flag("retries", "Number of times to retry AWS API calls in case of errors.").Default("10").Int()
)

func init() {
	kingpin.Parse()
	if *debug {
		log.SetLevel(log.DebugLevel)
	}
}

func getInstances(sess *session.Session, region string, instCh chan *instance) {
	ec2Svc := ec2.New(sess, aws.NewConfig().WithRegion(region))

	start := time.Now()
	ec2instances, err := ec2Svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String("running"), aws.String("pending")},
			},
		},
	})
	if err != nil {
		log.Error("Failed to query EC2 service in region %s: %v", region, err)
	}
	end := time.Now()

	count := 0
	for _, r := range ec2instances.Reservations {
		for _, i := range r.Instances {
			inst := newInstance(i)
			instCh <- &inst
			count++
		}
	}
	duration := end.Sub(start)
	log.Debugf("Queried region %s and got %d instances in %v", region, count, duration)
}

func main() {
	var insts instances
	instCh := make(chan *instance)
	var wg sync.WaitGroup

	start := time.Now()
	sess, err := session.NewSessionWithOptions(session.Options{
		Config:            *aws.NewConfig().WithMaxRetries(*retries),
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		log.Fatalf("Unable to create AWS session: %v", err)
	}

	// Query EC2 endpoints in each region in parallel
	for _, region := range *regions {
		wg.Add(1)
		go func(region string) {
			getInstances(sess, region, instCh)
			wg.Done()
		}(region)
	}
	// Close channel after all EC2 queries have returned
	go func() {
		wg.Wait()
		close(instCh)
	}()
	// Build up a slice of instances as they are returned to us
	for {
		inst, ok := <-instCh
		if !ok {
			break
		}
		insts = append(insts, inst)
	}

	// Sort and print instances that were returned to us
	if len(insts) > 0 {
		insts.sort()
		insts.printTable()
	}

	if *debug {
		end := time.Now()
		duration := end.Sub(start)
		log.Debugf("Queried %d regions for %d instances in %v", len(*regions), len(insts), duration)
	}
}
