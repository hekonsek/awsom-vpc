# Awsom VPC - easy AWS VPC for Go

Awsom VPC allows you to create AWS VPCs based on 
[Practical VPC Design](https://medium.com/aws-activate-startup-blog/practical-vpc-design-8412e1a18dcc)
practices. Also makes working with VPCs easier by providing human-friendly functions.

## Usage

Create new VPC:

```
import "github.com/hekonsek/awsom-vpc"
...
vpcName := "staging"
err := NewVpcBuilder(vpcName).Create()
```



## License

This project is distributed under Apache 2.0 license.