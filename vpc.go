package awsomvpc

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/go-errors/errors"
	"github.com/hekonsek/awsom-session"
	"strings"
)

// Structs

type VpcBuilder struct {
	Name      string
	CidrBlock string
	Subnets   []Subnet
}

type Subnet struct {
	Cidr             string
	AvailabilityZone string
}

func NewVpcBuilder(name string) *VpcBuilder {
	return &VpcBuilder{
		Name:      name,
		CidrBlock: "10.0.0.0/16",
		Subnets: []Subnet{
			{Cidr: "10.0.0.0/18", AvailabilityZone: "us-east-1a"},
			{Cidr: "10.0.64.0/18", AvailabilityZone: "us-east-1b"},
			{Cidr: "10.0.128.0/18", AvailabilityZone: "us-east-1c"},
		},
	}
}

func (vpc *VpcBuilder) Create() error {
	ec2Service, err := NewEc2Service()
	if err != nil {
		return err
	}

	vpcCreated, err := ec2Service.CreateVpc(&ec2.CreateVpcInput{
		CidrBlock: aws.String(vpc.CidrBlock),
	})
	if err != nil {
		return err
	}
	_, err = ec2Service.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{vpcCreated.Vpc.VpcId},
		Tags: []*ec2.Tag{
			&(ec2.Tag{
				Key:   aws.String("Name"),
				Value: aws.String(vpc.Name),
			}),
		},
	})
	if err != nil {
		return err
	}

	for _, subnet := range vpc.Subnets {
		subnetResult, err := ec2Service.CreateSubnet(&ec2.CreateSubnetInput{
			VpcId:            vpcCreated.Vpc.VpcId,
			CidrBlock:        aws.String(subnet.Cidr),
			AvailabilityZone: aws.String(subnet.AvailabilityZone),
		})
		if err != nil {
			return err
		}
		_, err = ec2Service.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{subnetResult.Subnet.SubnetId},
			Tags: []*ec2.Tag{
				&(ec2.Tag{
					Key:   aws.String("Name"),
					Value: aws.String(vpc.Name + "-subnet"),
				}),
			},
		})
		if err != nil {
			return err
		}
	}

	gatewayResult, err := ec2Service.CreateInternetGateway(&ec2.CreateInternetGatewayInput{})
	if err != nil {
		return err
	}
	_, err = ec2Service.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{gatewayResult.InternetGateway.InternetGatewayId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(vpc.Name + "-gateway"),
			},
		},
	})
	if err != nil {
		return err
	}
	_, err = ec2Service.AttachInternetGateway(&ec2.AttachInternetGatewayInput{
		VpcId:             vpcCreated.Vpc.VpcId,
		InternetGatewayId: gatewayResult.InternetGateway.InternetGatewayId,
	})
	if err != nil {
		return err
	}

	routeTableResults, err := ec2Service.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{vpcCreated.Vpc.VpcId},
			},
		},
	})
	if err != nil {
		return err
	}

	_, err = ec2Service.CreateRoute(&ec2.CreateRouteInput{
		GatewayId:            gatewayResult.InternetGateway.InternetGatewayId,
		RouteTableId:         routeTableResults.RouteTables[0].RouteTableId,
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
	})
	if err != nil {
		return err
	}

	subnets, err := ec2Service.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpcId"),
				Values: []*string{vpcCreated.Vpc.VpcId},
			},
		},
	})
	if err != nil {
		return err
	}
	for _, subnet := range subnets.Subnets {
		_, err = ec2Service.AssociateRouteTable(&ec2.AssociateRouteTableInput{
			RouteTableId: routeTableResults.RouteTables[0].RouteTableId,
			SubnetId:     subnet.SubnetId,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func VpcExistsByName(name string) (bool, error) {
	if name == "" {
		return false, errors.New("name of VPC cannot be empty")
	}

	ec2Service, err := NewEc2Service()
	if err != nil {
		return false, err
	}

	vpcs, err := ec2Service.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []*string{aws.String(name)},
			},
		},
	})
	if err != nil {
		return false, err
	}
	return len(vpcs.Vpcs) > 0, nil
}

func VpcId(name string) (string, error) {
	if name == "" {
		return "", errors.New("name of VPC cannot be empty")
	}

	ec2Service, err := NewEc2Service()
	if err != nil {
		return "", err
	}

	vpcs, err := ec2Service.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []*string{aws.String(name)},
			},
		},
	})
	if err != nil {
		return "", err
	}

	if len(vpcs.Vpcs) > 0 {
		return *vpcs.Vpcs[0].VpcId, nil
	} else {
		return "", errors.New("cannot find VPC with name " + name)
	}
}

func SubnetId(name string) (string, error) {
	ec2Service, err := NewEc2Service()
	if err != nil {
		return "", err
	}

	subnets, err := ec2Service.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []*string{aws.String(name + "-subnet")},
			},
		},
	})
	if err != nil {
		return "", err
	}

	if len(subnets.Subnets) > 0 {
		return *subnets.Subnets[0].SubnetId, nil
	} else {
		return "", nil
	}
}

func DeleteVpc(name string) error {
	ec2Service, err := NewEc2Service()
	if err != nil {
		return err
	}

	vpcId, err := VpcId(name)
	if err != nil {
		return err
	}

	gatewayResults, err := ec2Service.DescribeInternetGateways(&ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []*string{aws.String(name + "-gateway")},
			},
		},
	})
	if err != nil {
		return err
	}

	if len(gatewayResults.InternetGateways) == 1 {
		_, err = ec2Service.DetachInternetGateway(&ec2.DetachInternetGatewayInput{
			VpcId:             aws.String(vpcId),
			InternetGatewayId: gatewayResults.InternetGateways[0].InternetGatewayId,
		})
		if err != nil {
			return err
		}

		_, err = ec2Service.DeleteInternetGateway(&ec2.DeleteInternetGatewayInput{
			InternetGatewayId: gatewayResults.InternetGateways[0].InternetGatewayId,
		})
		if err != nil {
			return err
		}
	}

	routeTableResults, err := ec2Service.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpcId)},
			},
		},
	})
	if err != nil {
		return err
	}

	for _, routeTable := range routeTableResults.RouteTables {
		for _, route := range routeTable.Routes {
			if !strings.HasSuffix(*route.DestinationCidrBlock, "/16") {
				_, err = ec2Service.DeleteRoute(&ec2.DeleteRouteInput{
					RouteTableId:         routeTable.RouteTableId,
					DestinationCidrBlock: route.DestinationCidrBlock,
				})
				if err != nil {
					return err
				}
			}
		}
		for _, association := range routeTable.Associations {
			if !*association.Main {
				_, err = ec2Service.DisassociateRouteTable(&ec2.DisassociateRouteTableInput{
					AssociationId: association.RouteTableAssociationId,
				})
				if err != nil {
					return err
				}
			}
		}
	}

	subnets, err := VpcSubnetsByName(name)
	if err != nil {
		return err
	}
	for _, subnet := range subnets {
		_, err = ec2Service.DeleteSubnet(&ec2.DeleteSubnetInput{
			SubnetId: aws.String(subnet),
		})
		if err != nil {
			return err
		}
	}

	_, err = ec2Service.DeleteVpc(&ec2.DeleteVpcInput{
		VpcId: aws.String(vpcId),
	})
	if err != nil {
		return err
	}

	return nil
}

func VpcSubnetsByName(vpcName string) ([]string, error) {
	ec2Service, err := NewEc2Service()
	if err != nil {
		return nil, err
	}

	vpcId, err := VpcId(vpcName)
	if err != nil {
		return nil, err
	}

	subnets, err := ec2Service.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpcId)},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	subnetsIds := []string{}
	for _, subnet := range subnets.Subnets {
		subnetsIds = append(subnetsIds, *subnet.SubnetId)
	}
	return subnetsIds, nil
}

// Services

// NewEc2Service creates new instance of AWS SDK EC2 service. Relies on "github.com/hekonsek/awsom-session" for
// session conventions.
func NewEc2Service() (*ec2.EC2, error) {
	sess, err := awsom_session.NewSession()
	if err != nil {
		return nil, err
	}
	return ec2.New(sess), nil
}
