package sweepers

import (
	"errors"
	"fmt"

	container "github.com/scaleway/scaleway-sdk-go/api/container/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/logger"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func SweepTrigger(scwClient *scw.Client, region scw.Region) error {
	containerAPI := container.NewAPI(scwClient)
	logger.Warningf("sweeper: destroying container triggers in %q", region)

	listTriggers, err := containerAPI.ListTriggers(&container.ListTriggersRequest{
		Region: region,
	}, scw.WithAllPages())
	if err != nil {
		return fmt.Errorf("error listing triggers: %w", err)
	}

	errs := []error(nil)
	for _, trigger := range listTriggers.Triggers {
		_, err := containerAPI.DeleteTrigger(&container.DeleteTriggerRequest{
			TriggerID: trigger.ID,
			Region:    region,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("error deleting trigger %s: %w", trigger.ID, err))
		}
	}

	return errors.Join(errs...)
}

func SweepContainer(scwClient *scw.Client, region scw.Region) error {
	containerAPI := container.NewAPI(scwClient)
	logger.Warningf("sweeper: destroying containers in %q", region)

	listNamespaces, err := containerAPI.ListContainers(&container.ListContainersRequest{
		Region: region,
	}, scw.WithAllPages())
	if err != nil {
		return fmt.Errorf("error listing containers: %w", err)
	}

	errs := []error(nil)
	for _, cont := range listNamespaces.Containers {
		_, err := containerAPI.DeleteContainer(&container.DeleteContainerRequest{
			ContainerID: cont.ID,
			Region:      region,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("error deleting container %s: %w", cont.ID, err))
		}
	}

	return errors.Join(errs...)
}

func SweepNamespace(scwClient *scw.Client, region scw.Region) error {
	containerAPI := container.NewAPI(scwClient)
	logger.Warningf("sweeper: destroying container namespaces in %q", region)

	listNamespaces, err := containerAPI.ListNamespaces(&container.ListNamespacesRequest{
		Region: region,
	}, scw.WithAllPages())
	if err != nil {
		return fmt.Errorf("error listing namespaces: %w", err)
	}

	errs := []error(nil)
	for _, ns := range listNamespaces.Namespaces {
		_, err := containerAPI.DeleteNamespace(&container.DeleteNamespaceRequest{
			NamespaceID: ns.ID,
			Region:      region,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("error deleting namespace %s: %w", ns.ID, err))
		}
	}

	return errors.Join(errs...)
}

func SweepAllLocalities(scwClient *scw.Client) error {
	errs := []error(nil)
	for _, region := range (&container.API{}).Regions() {
		err := SweepTrigger(scwClient, region)
		if err != nil {
			errs = append(errs, err)
		}
		err = SweepContainer(scwClient, region)
		if err != nil {
			errs = append(errs, err)
		}
		err = SweepNamespace(scwClient, region)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
