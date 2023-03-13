package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"

	"github.com/andrey-berenda/perfect-driver/internal/pkg/cloudformation"
	"github.com/andrey-berenda/perfect-driver/internal/pkg/ptr"
)

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	cloudformation.NewStack(app, "Stack", &awscdk.StackProps{
		Env: &awscdk.Environment{
			Account: ptr.Of("894081577876"),
			Region:  ptr.Of("us-east-1"),
		},
	})

	app.Synth(nil)
}
