# awspause

awspause is a small go-lang programm, that helps you to start/stop expensive instances on aws that are only partially needed (like dev stacks e.g.) but need to be non-ephemeral.

## Prerequesites

Setup and configure awscli to use profiles from ~/.aws/config and ~/.aws/credentials.

All instances, that are allowed to be stopped and started need to be tagged with:

```
Ephemeral=False
Pausable=True
```

Also you need to "Enable termination protection" on the instance and uncheck "Delete on termination" on the root device.

Using terraform, the following definition might help:

```
resource "aws_instance" "web" {
  ami           = "__AMI_ID__"
  instance_type = "t2.micro"

  private_ip = "__PRIVATE_IP__"
  subnet_id  = "__SUBNET_ID__"

  key_name = "__KEY_NAME__"

  vpc_security_group_ids = ["__SECURITY_GROUP_ID__"]

  disable_api_termination = true

  tags = {
    Ephemeral = "False"
    Pausable = "True"
  }

  root_block_device {
    delete_on_termination = false
  }

}
```

## Usage

```
awspause command [options]
```

### commands

  * start - start all pausable machines
  * stop - stop all pausable machines

### options

  * verbose,v - print return values and debugging information
  * profile,p - select the profile to use
  * tags,t - add additional tags, to filter for (e.g. --tags="Name=fu,Environment=bar")
  * ids,i - add subset of instance IDs, to iterate over (e.g. --ids=i-1234*************,i-5678*************)

## Alternative

Alternatively you can start/stop instances by awscli as well, without the need of installing this little go helper.

### starting all pausable instances
```
aws ec2 describe-instances --filters Name=tag:Ephemeral,Values="False" Name=tag:Pausable,Values="True" | jq ".Reservations[].Instances[].InstanceId" -r | xargs aws ec2 start-instances --instance-ids
```

### stopping all pausable Instances
```
aws ec2 describe-instances --filters Name=tag:Ephemeral,Values="False" Name=tag:Pausable,Values="True" | jq ".Reservations[].Instances[].InstanceId" -r | xargs aws ec2 stop-instances --instance-ids
```
