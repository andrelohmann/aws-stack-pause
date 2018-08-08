package main

import (
  "fmt"
  "os"
  "strings"
  flag "github.com/ogier/pflag"
  aws "github.com/aws/aws-sdk-go/aws"
  awserr "github.com/aws/aws-sdk-go/aws/awserr"
  awssession "github.com/aws/aws-sdk-go/aws/session"
  awsec2 "github.com/aws/aws-sdk-go/service/ec2"
)

// flags
var (
  profile   string
  verbose   bool
  tags      string
  ids       string
  session   *awssession.Session
  service   *awsec2.EC2
  filter    *awsec2.DescribeInstancesInput
  instances []*string
)

func main() {

  // if no parameters are set
  if len(os.Args) < 2 {
    printUsage()
  }

  switch os.Args[1] {
    case "start":
      startInstances()
    case "stop":
      stopInstances()
    default:
      printUsage()
  }
}

func init() {
  flag.StringVarP(&profile, "profile", "p", "default", "Select the profile to use")
  flag.BoolVarP(&verbose, "verbose", "v", false, "Print return value")
  flag.StringVarP(&tags, "tags", "t", "", "Filter by additional tags (e.g. --tags=\"Name=fu,Environment=bar\"")
  flag.StringVarP(&ids, "ids", "i", "", "Filter by instance ids (e.g. --ids=i-1234*************,i-5678*************")
  flag.Parse()
  session = loadSession()
  service = loadService()
  filter = &awsec2.DescribeInstancesInput{
    Filters: []*awsec2.Filter{
      &awsec2.Filter{
        Name: aws.String("tag:Ephemeral"),
        Values: []*string{
          aws.String("False"),
        },
      },
      &awsec2.Filter{
        Name: aws.String("tag:Pausable"),
        Values: []*string{
          aws.String("True"),
        },
      },
      &awsec2.Filter{
        Name: aws.String("instance-state-name"),
        Values: []*string{
          aws.String("pending"),
          aws.String("running"),
          aws.String("shutting-down"),
          aws.String("stopping"),
          aws.String("stopped"),
        },
      },
    },
  }
  additionalFilters(filter)
  loadInstances()
}

func loadSession() *awssession.Session {
  // Force enable Shared Config support
  sess, err := awssession.NewSessionWithOptions(awssession.Options{
    SharedConfigState: awssession.SharedConfigEnable,
    Profile: profile,
  })

  if err != nil {
    fmt.Println("Error creating session ", err)
    os.Exit(1)
  }

  return sess
}

func loadService() *awsec2.EC2 {
  svc := awsec2.New(session)

  return svc
}

func additionalFilters(filter *awsec2.DescribeInstancesInput) {
  // If filters are set
  if len(tags) > 0 {
    for _, _tag := range strings.Split(tags, ","){
      tag := strings.Split(_tag, "=")
      if verbose {
        fmt.Println("Tag: ", tag[0], ", Value: ", tag[1])
      }
      filter.Filters = append(filter.Filters, &awsec2.Filter{
        Name: aws.String("tag:"+tag[0]),
        Values: []*string{
          aws.String(tag[1]),
        },
      })
    }
  }

  if len(ids) > 0 {
    var _ids []*string
    for _, _id := range strings.Split(ids, ","){
      _ids = append(_ids, &_id)
    }

    filter.InstanceIds = _ids
  }
}

func loadInstances() {
  result, err := service.DescribeInstances(filter)
  if err != nil {
    fmt.Println("Error on DescribeInstances", err)
    os.Exit(1)
  } else {
    if verbose {
      fmt.Println("Success on DescribeInstances", result)
    }

    for idx, res := range result.Reservations {
      if verbose {
        fmt.Println("  > Reservation Id", *res.ReservationId, " Num Instances: ", len(res.Instances))
      }

      for _, inst := range result.Reservations[idx].Instances {
        if verbose {
          fmt.Println("    - Instance ID: ", *inst.InstanceId)
        }
        instances = append(instances, inst.InstanceId)
      }
    }
  }
}

func stopInstances() {
  if len(instances) > 0 {
    input := &awsec2.StopInstancesInput{
      InstanceIds: instances,
      DryRun: aws.Bool(true),
    }

    result, err := service.StopInstances(input)
    awsErr, ok := err.(awserr.Error)

    if ok && awsErr.Code() == "DryRunOperation" {
      input.DryRun = aws.Bool(false)
      result, err = service.StopInstances(input)

      if err != nil {
        fmt.Println("Error", err)
        os.Exit(1)
      } else {
        if verbose {
          fmt.Println("Success", result.StoppingInstances)
        } else {
          fmt.Printf("Success: stopping %v instance(s)\n", len(instances))
        }
      }
    } else {
      fmt.Println("Error", err)
      os.Exit(1)
    }
  } else {
    fmt.Println("No instances found")
  }
  os.Exit(0)
}

func startInstances() {
  if len(instances) > 0 {
    input := &awsec2.StartInstancesInput{
      InstanceIds: instances,
      DryRun: aws.Bool(true),
    }

    result, err := service.StartInstances(input)
    awsErr, ok := err.(awserr.Error)

    if ok && awsErr.Code() == "DryRunOperation" {
      // Let's now set dry run to be false. This will allow us to start the instances
      input.DryRun = aws.Bool(false)
      result, err = service.StartInstances(input)
      if err != nil {
        fmt.Println("Error", err)
        os.Exit(1)
      } else {
        if verbose {
          fmt.Println("Success", result.StartingInstances)
        } else {
          fmt.Printf("Success: starting %v instance(s)\n", len(instances))
        }
      }
    } else { // This could be due to a lack of permissions
      fmt.Println("Error", err)
      os.Exit(1)
    }
  } else {
    fmt.Println("No instances found")
  }
  os.Exit(0)
}

func printUsage() {
  fmt.Println("awspause helps you stop/start (pause) expensive EC2 Instances on AWS, that need to be non-ephemeral but shouldn't be running all the time.")
  fmt.Println("Tag all Instances with Name=Ephemeral,Values=\"False\" and Name=Pausable,Values=\"True\", enable termination protection and set DeleteOnTermination to false.")
  fmt.Println("")
  fmt.Println("Usage:")
  fmt.Println("         awspause command [options]")
  fmt.Println("")
  fmt.Println("The commands are:")
  fmt.Println("         start   start all paused machines")
  fmt.Println("         stop    stop all pausable machines")
  fmt.Println("")
  fmt.Println("Options:")
  flag.PrintDefaults()
  os.Exit(1)
}
