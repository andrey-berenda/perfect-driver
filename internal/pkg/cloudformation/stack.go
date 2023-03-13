package cloudformation

import (
	_ "embed"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsautoscaling"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsrds"
	"github.com/aws/aws-cdk-go/awscdklambdagoalpha/v2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"

	"github.com/andrey-berenda/perfect-driver/internal/pkg/ptr"
)

//go:embed cfn/cfn-hup.conf
var cfnHupConf string

//go:embed cfn/cfn-hup.service
var cfnHupService string

//go:embed cloudwatch/config.json
var cloudWatchAgentJSON string

//go:embed cloudwatch/agent-auto-reloader.conf
var cloudWatchAgentAutoReloaderConf string

//go:embed cloudwatch/install-agent.sh
var installCloudWatchAgentSh string

//go:embed bot/get-status.sh
var botGetStatus string

//go:embed bot/bot.service
var botService string

//go:embed bot/logrotate
var logrotateFile string

//go:embed bot/restart.sh
var restartBot string

//go:embed bot/start.sh
var startBot string

func NewStack(scope constructs.Construct, id string, props *awscdk.StackProps) awscdk.Stack {
	stack := awscdk.NewStack(scope, &id, props)

	publicKeyMaterial := awscdk.NewCfnParameter(stack, ptr.Of("PublicKeyMaterial"), &awscdk.CfnParameterProps{
		Description: ptr.Of("Ssh public key"),
		Type:        ptr.Of("String"),
	})

	dbUsername := awscdk.NewCfnParameter(stack, ptr.Of("DBUsername"), &awscdk.CfnParameterProps{
		NoEcho:      ptr.Of(true),
		Description: ptr.Of("PostgreSQL database username"),
		Type:        ptr.Of("String"),
	})

	dbPassword := awscdk.NewCfnParameter(stack, ptr.Of("DBPassword"), &awscdk.CfnParameterProps{
		NoEcho:      ptr.Of(true),
		Description: ptr.Of("PostgreSQL database password"),
		Type:        ptr.Of("String"),
	})

	defaultVpc := awsec2.Vpc_FromLookup(stack, ptr.Of("VPC"), &awsec2.VpcLookupOptions{
		IsDefault: ptr.Of(true),
		Region:    stack.Region(),
	})

	keyPair := awsec2.NewCfnKeyPair(stack, ptr.Of("EC2KeyPair"), &awsec2.CfnKeyPairProps{
		KeyName:           ptr.Of("EC2KeyName"),
		PublicKeyMaterial: publicKeyMaterial.ValueAsString(),
	})

	ec2Role := awsiam.NewRole(stack, ptr.Of("EC2Role"), &awsiam.RoleProps{
		RoleName:  ptr.Of("EC2IAMRole"),
		AssumedBy: awsiam.NewServicePrincipal(ptr.Of("ec2.amazonaws.com"), nil),
		Path:      ptr.Of("/"),
		InlinePolicies: &map[string]awsiam.PolicyDocument{
			"WriteCloudWatchLogs": awsiam.NewPolicyDocument(&awsiam.PolicyDocumentProps{
				Statements: &[]awsiam.PolicyStatement{
					awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
						Effect: awsiam.Effect_ALLOW,
						Actions: &[]*string{
							ptr.Of("logs:CreateLogGroup"),
							ptr.Of("logs:CreateLogStream"),
							ptr.Of("logs:PutLogEvents"),
							ptr.Of("logs:DescribeLogStreams"),
						},
						Resources: &[]*string{ptr.Of("*")},
					}),
				},
			}),
			"ReadAppConfig": awsiam.NewPolicyDocument(&awsiam.PolicyDocumentProps{
				Statements: &[]awsiam.PolicyStatement{
					awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
						Effect: awsiam.Effect_ALLOW,
						Actions: &[]*string{
							ptr.Of("appconfig:GetLatestConfiguration"),
							ptr.Of("appconfig:StartConfigurationSession"),
						},
						Resources: &[]*string{
							ptr.Of("*"),
						},
					}),
				},
			}),
		},
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(ptr.Of("AmazonSSMManagedInstanceCore")),
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(ptr.Of("AmazonRDSReadOnlyAccess")),
		},
	})

	awsiam.NewCfnInstanceProfile(stack, ptr.Of("EC2InstanceProfile"), &awsiam.CfnInstanceProfileProps{
		InstanceProfileName: ec2Role.RoleName(),
		Roles:               &[]*string{ec2Role.RoleName()},
	})

	ec2Sg := awsec2.NewSecurityGroup(stack, ptr.Of("EC2SecurityGroup"), &awsec2.SecurityGroupProps{
		Vpc:               defaultVpc,
		SecurityGroupName: ptr.Of("EC2SecurityGroup"),
	})

	ec2Sg.AddIngressRule(
		awsec2.Peer_AnyIpv4(),
		awsec2.NewPort(&awsec2.PortProps{
			StringRepresentation: ptr.Of("ec2"),
			Protocol:             awsec2.Protocol_TCP,
			FromPort:             jsii.Number(22),
			ToPort:               jsii.Number(22),
		}),
		nil,
		nil,
	)

	dbSg := awsec2.NewSecurityGroup(stack, ptr.Of("DBSecurityGroup"), &awsec2.SecurityGroupProps{
		Vpc:               defaultVpc,
		SecurityGroupName: ptr.Of("DBSecurityGroup"),
	})
	dbSg.AddIngressRule(
		awsec2.Peer_AnyIpv4(),
		awsec2.NewPort(&awsec2.PortProps{
			StringRepresentation: ptr.Of("db"),
			Protocol:             awsec2.Protocol_TCP,
			FromPort:             jsii.Number(5432),
			ToPort:               jsii.Number(5432),
		}),
		nil,
		nil,
	)

	autoScalingGroup := awsautoscaling.NewAutoScalingGroup(stack, ptr.Of("EC2Instance"), &awsautoscaling.AutoScalingGroupProps{
		InstanceType: awsec2.NewInstanceType(ptr.Of("t2.micro")),
		MachineImage: awsec2.MachineImage_LatestAmazonLinux(&awsec2.AmazonLinuxImageProps{
			Generation: awsec2.AmazonLinuxGeneration_AMAZON_LINUX_2,
		}),
		MaxCapacity: jsii.Number(1),
		MinCapacity: jsii.Number(1),
		Role:        ec2Role,
		Vpc:         defaultVpc,
		UserData:    awsec2.UserData_Custom(awscdk.Fn_Sub(ptr.Of(installCloudWatchAgentSh), nil)),
		KeyName:     keyPair.KeyName(),
		Init: awsec2.CloudFormationInit_FromConfigSets(
			&awsec2.ConfigSetProps{
				ConfigSets: &map[string]*[]*string{
					"default": {
						ptr.Of("01_setupCfnHup"),
						ptr.Of("02_setupBot"),
						ptr.Of("03_config-amazon-cloudwatch-agent"),
						ptr.Of("04_restart_amazon-cloudwatch-agent"),
					},
					"UpdateCloudWatch": {
						ptr.Of("03_config-amazon-cloudwatch-agent"),
						ptr.Of("04_restart_amazon-cloudwatch-agent"),
					},
				},
				Configs: &map[string]awsec2.InitConfig{
					"03_config-amazon-cloudwatch-agent": awsec2.NewInitConfig(&[]awsec2.InitElement{
						awsec2.InitFile_FromString(
							ptr.Of("/opt/aws/amazon-cloudwatch-agent/etc/amazon-config.json"),
							awscdk.Fn_Sub(ptr.Of(cloudWatchAgentJSON), nil),
							&awsec2.InitFileOptions{
								Group: ptr.Of("root"),
								Owner: ptr.Of("root"),
								Mode:  ptr.Of("000400"),
							},
						),
					}),
					"04_restart_amazon-cloudwatch-agent": awsec2.NewInitConfig(&[]awsec2.InitElement{
						awsec2.InitCommand_ShellCommand(
							ptr.Of("/opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-ctl -a stop"),
							nil,
						),
						awsec2.InitCommand_ShellCommand(
							ptr.Of("/opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-ctl -a fetch-config -m ec2 -c file:/opt/aws/amazon-cloudwatch-agent/etc/amazon-config.json -s"),
							nil,
						),
					}),
					"01_setupCfnHup": awsec2.NewInitConfig(&[]awsec2.InitElement{
						awsec2.InitFile_FromString(
							ptr.Of("/etc/cfn/cfn-hup.conf"),
							awscdk.Fn_Sub(ptr.Of(cfnHupConf), nil),
							&awsec2.InitFileOptions{
								Group: ptr.Of("root"),
								Owner: ptr.Of("root"),
								Mode:  ptr.Of("000400"),
							},
						),
						awsec2.InitFile_FromString(
							ptr.Of("/etc/cfn/hooks.d/amazon-agent-auto-reloader.conf"),
							awscdk.Fn_Sub(ptr.Of(cloudWatchAgentAutoReloaderConf), nil),
							&awsec2.InitFileOptions{
								Group: ptr.Of("root"),
								Owner: ptr.Of("root"),
								Mode:  ptr.Of("000400"),
							},
						),
						awsec2.InitFile_FromString(
							ptr.Of("/lib/systemd/system/cfn-hup.service"),
							ptr.Of(cfnHupService),
							&awsec2.InitFileOptions{
								Group: ptr.Of("root"),
								Owner: ptr.Of("root"),
								Mode:  ptr.Of("000400"),
							},
						),
						awsec2.InitCommand_ArgvCommand(
							&[]*string{
								ptr.Of("systemctl"),
								ptr.Of("enable"),
								ptr.Of("cfn-hup.service"),
							},
							nil,
						),
						awsec2.InitCommand_ArgvCommand(
							&[]*string{
								ptr.Of("systemctl"),
								ptr.Of("start"),
								ptr.Of("cfn-hup.service"),
							},
							nil,
						),
					}),
					"02_setupBot": awsec2.NewInitConfig(&[]awsec2.InitElement{
						awsec2.InitFile_FromString(
							ptr.Of("/etc/logrotate.d/bot"),
							awscdk.Fn_Sub(ptr.Of(logrotateFile), nil),
							&awsec2.InitFileOptions{
								Group: ptr.Of("root"),
								Owner: ptr.Of("root"),
								Mode:  ptr.Of("000644"),
							},
						),
						awsec2.InitFile_FromString(
							ptr.Of("/usr/local/bin/get-status"),
							ptr.Of(botGetStatus),
							&awsec2.InitFileOptions{
								Group: ptr.Of("root"),
								Owner: ptr.Of("root"),
								Mode:  ptr.Of("000655"),
							},
						),
						awsec2.InitFile_FromString(
							ptr.Of("/usr/local/bin/restart-bot"),
							ptr.Of(restartBot),
							&awsec2.InitFileOptions{
								Group: ptr.Of("root"),
								Owner: ptr.Of("root"),
								Mode:  ptr.Of("000655"),
							},
						),
						awsec2.InitFile_FromString(
							ptr.Of("/usr/local/bin/start-bot"),
							ptr.Of(startBot),
							&awsec2.InitFileOptions{
								Group: ptr.Of("root"),
								Owner: ptr.Of("root"),
								Mode:  ptr.Of("000655"),
							},
						),
						awsec2.InitFile_FromString(
							ptr.Of("/lib/systemd/system/bot.service"),
							ptr.Of(botService),
							&awsec2.InitFileOptions{
								Group: ptr.Of("root"),
								Owner: ptr.Of("root"),
								Mode:  ptr.Of("000400"),
							},
						),
						awsec2.InitCommand_ArgvCommand(
							&[]*string{
								ptr.Of("systemctl"),
								ptr.Of("enable"),
								ptr.Of("bot.service"),
							},
							nil,
						),
					}),
				},
			},
		),
		Signals: awsautoscaling.Signals_WaitForAll(&awsautoscaling.SignalsOptions{
			MinSuccessPercentage: jsii.Number(100),
		}),
	})

	autoScalingGroup.AddSecurityGroup(ec2Sg)

	awsrds.NewCfnDBInstance(stack, ptr.Of("DBInstance"), &awsrds.CfnDBInstanceProps{
		AllocatedStorage:     ptr.Of("20"),
		PubliclyAccessible:   ptr.Of(true),
		MasterUsername:       dbUsername.ValueAsString(),
		MasterUserPassword:   dbPassword.ValueAsString(),
		VpcSecurityGroups:    &[]*string{dbSg.SecurityGroupId()},
		EngineVersion:        ptr.Of("14.6"),
		Engine:               ptr.Of("postgres"),
		DbInstanceClass:      ptr.Of("db.t3.micro"),
		DbInstanceIdentifier: ptr.Of("db"),
	})

	awscdklambdagoalpha.NewGoFunction(stack, ptr.Of("NotifyAboutDeployLambda"), &awscdklambdagoalpha.GoFunctionProps{
		FunctionName: ptr.Of("TriggerForFormInSite"),
		Entry:        ptr.Of("cmd/lambda"),
	})

	return stack
}
