package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/VaibhavPage/tekton-cd-trigger/proto"
	"github.com/ghodss/yaml"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	batch "k8s.io/api/batch/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type CronJobTrigger struct {
	client *rest.RESTClient
}

// FetchResource fetches the resource to be triggered.
func (t *CronJobTrigger) FetchResource(ctx context.Context, in *proto.FetchResourceRequest) (*proto.FetchResourceResponse, error) {
	var in_resource map[string]string
	if err := yaml.Unmarshal(in.Resource, &in_resource); err != nil {
		return nil, err
	}

	namespace := in_resource["namespace"]
	cronjobName := in_resource["cronjob"]
	log.Info().Str("name", cronjobName).Str("namespace", namespace).Msg("Fetching CronJob")

	// Fetch CronJob
	result := t.client.Get().Resource("cronjobs").Namespace(namespace).Name(cronjobName).Do(ctx)
	if result.Error() != nil {
		return nil, result.Error()
	}

	// Parse CronJob
	cronjob := new(batch.CronJob)
	if err := result.Into(cronjob); err != nil {
		return nil, result.Error()
	}

	// Create Job
	job := batch.Job{
		TypeMeta: v1.TypeMeta{
			Kind:       "Job",
			APIVersion: cronjob.TypeMeta.APIVersion,
		},
		Spec: cronjob.Spec.JobTemplate.Spec,
		ObjectMeta: v1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: fmt.Sprintf("%s-argo-events-", cronjob.ObjectMeta.Name),
		},
	}

	// Marshal Job
	resource, err := yaml.Marshal(job)
	if err != nil {
		return nil, err
	}

	return &proto.FetchResourceResponse{
		Resource: resource,
	}, nil
}

// Execute executes the requested trigger resource.
func (t *CronJobTrigger) Execute(ctx context.Context, in *proto.ExecuteRequest) (*proto.ExecuteResponse, error) {
	job := new(batch.Job)
	if err := yaml.Unmarshal(in.Resource, job); err != nil {
		return nil, err
	}

	namespace := job.ObjectMeta.Namespace
	log.Info().Str("namespace", namespace).Str("name", job.ObjectMeta.GenerateName).Msg("Creating Job")
	result := t.client.Post().Resource("jobs").Namespace(namespace).Body(job).Do(ctx)
	if result.Error() != nil {
		return nil, result.Error()
	}

	return &proto.ExecuteResponse{
		Response: []byte("success"),
	}, nil
}

// ApplyPolicy applies policies on the trigger execution result.
func (t *CronJobTrigger) ApplyPolicy(ctx context.Context, in *proto.ApplyPolicyRequest) (*proto.ApplyPolicyResponse, error) {
	return &proto.ApplyPolicyResponse{
		Success: true,
		Message: "success",
	}, nil
}

func createClient() (*rest.RESTClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	config.GroupVersion = &batch.SchemeGroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	httpClient, err := rest.HTTPClientFor(config)
	if err != nil {
		return nil, err
	}
	client, err := rest.RESTClientForConfigAndClient(config, httpClient)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func main() {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "9000"
	}

	client, err := createClient()
	if err != nil {
		panic(err)
	}

	trigger := &CronJobTrigger{client}
	log.Info().Str("port", port).Msg("starting trigger server")

	// Start server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		panic(err)
	}

	srv := grpc.NewServer()
	proto.RegisterTriggerServer(srv, trigger)
	if err := srv.Serve(lis); err != nil {
		panic(err)
	}
}
