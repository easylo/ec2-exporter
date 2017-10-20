package exporter

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	// "github.com/labstack/gommon/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "aws_ec2"
)

var (
	instanceState = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "instance_describe"),
		"Get information about aws ec2 instances and up time in hours",
		[]string{
			"instance_id",
			"instance_state_name",
			"instance_type",
			"image_id",
			"region",
			"instance_tags",
		},
		nil,
	)
)

// Exporter exports EC2 metrics
type Exporter struct {
	awsKey    string
	awsSecret string
	regions   []string
	mu        sync.RWMutex
}

// DataResult EC2 metrics
type DataResult struct {
	Region    string
	Instances []*ec2.Instance
}

// New returns an initialized Exporter.
// func New(awsKey string, awsSecret string, awsRegion []string) *Exporter {
func New(awsKey string, awsSecret string, regions []string) *Exporter {

	return &Exporter{
		awsKey:    awsKey,
		awsSecret: awsSecret,
		regions:   regions,
	}
}

// Round returns round float
func Round(f float64) float64 {
	return math.Floor(f + .5)
}

// Describe describes all the metrics ever exported by the NewRelic exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- instanceState
}

// Collect fetches the stats from configured NewRelic and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mu.Lock() // To protect metrics from concurrent collects.
	defer e.mu.Unlock()

	dataResults, err := getData(e)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	for _, dataResult := range dataResults {

		for _, instance := range dataResult.Instances {

			duration := time.Since(*instance.LaunchTime)
			value := Round(duration.Hours())

			InstanceID := *instance.InstanceId
			InstanceStateName := *instance.State.Name
			InstanceType := *instance.InstanceType
			ImageID := *instance.ImageId

			var InstanceTags []string
			for _, element := range instance.Tags {
				InstanceTags = append(InstanceTags, *element.Key+"="+*element.Value)
			}

			region := dataResult.Region

			ch <- prometheus.MustNewConstMetric(
				instanceState, prometheus.GaugeValue, value, InstanceID, InstanceStateName, InstanceType, ImageID, region, strings.Join(InstanceTags, ","),
			)
		}
	}
}

func getData(e *Exporter) ([]DataResult, error) {

	var data []DataResult

	regions := e.regions

	if len(regions) == 1 {

		var err error
		regions, err = fetchRegion(e)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	for _, region := range regions {
		var instances []*ec2.Instance
		sess := session.Must(session.NewSession(&aws.Config{
			Region:      aws.String(region),
			Credentials: credentials.NewStaticCredentials(e.awsKey, e.awsSecret, ""),
		}))

		ec2Svc := ec2.New(sess)
		params := &ec2.DescribeInstancesInput{}

		result, err := ec2Svc.DescribeInstances(params)
		// p := fmt.Println
		if err != nil {
			fmt.Println("Error", err)
		} else {
			// fmt.Printf("\n\n\nFetching instace details  for region: %s with criteria: %s**\n ", region, instanceCriteria)
			if len(result.Reservations) == 0 {
				// fmt.Printf("There is no instance for the for region %s with the matching Criteria:%s  \n", region, instanceCriteria)
			}
			for _, reservation := range result.Reservations {

				// fmt.Println("printing instance details.....")
				for _, instance := range reservation.Instances {
					instances = append(instances, instance)
				}

				// 	fmt.Println("instance id " + *instance.InstanceId)
				// 	fmt.Println("current State " + *instance.State.Name)
				// 	// fmt.Println("current DnsName " + *instance.Dns.Name)
				// 	fmt.Println("current ImageId " + *instance.ImageId)
				// 	fmt.Println("current InstanceType " + *instance.InstanceType)
				// 	// fmt.Println("current State code " + strconv.Itoa(*instance.State.Code))

				// 	p(*instance.LaunchTime)
				// 	fmt.Println("current PrivateDnsName " + *instance.PrivateDnsName)
				// 	fmt.Println("current PrivateIpAddress " + *instance.PrivateIpAddress)
				// 	// fmt.Println("current PublicDnsName " + *instance.PublicDnsName)
				// 	// fmt.Println("current PublicIpAddress " + *instance.PublicIpAddress)
				// 	// fmt.Println("current SpotInstanceRequestId " + *instance.SpotInstanceRequestId)

				// 	// for _, element := range instance.BlockDeviceMappings {
				// 	// 	fmt.Println("current Tags " + *element.DeviceName + ":" + *element.Ebs.VolumeId)
				// 	// }

				// 	for _, element := range instance.Tags {
				// 		// index is the index where we are
				// 		// element is the element from someSlice for where we are
				// 		fmt.Println("current Tags " + *element.Key + ":" + *element.Value)
				// 	}
				// 	// fmt.Println("current Tags " + *instance.Tags)
				// 	fmt.Println("####################################################")
				// }
			}
			dataResult := DataResult{
				Region:    region,
				Instances: instances,
			}

			data = append(data, dataResult)

			// fmt.Printf("done for region %s **** \n", region)

		}
	}
	return data, nil
}

func fetchRegion(e *Exporter) ([]string, error) {
	awsSession := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String("us-west-1"),
		Credentials: credentials.NewStaticCredentials(e.awsKey, e.awsSecret, ""),
	}))

	svc := ec2.New(awsSession)
	awsRegions, err := svc.DescribeRegions(&ec2.DescribeRegionsInput{})
	if err != nil {
		return nil, err
	}

	regions := make([]string, 0, len(awsRegions.Regions))
	for _, region := range awsRegions.Regions {
		regions = append(regions, *region.RegionName)
	}

	return regions, nil
}
