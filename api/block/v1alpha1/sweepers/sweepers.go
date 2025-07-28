package sweepers

import (
	"errors"
	"fmt"

	block "github.com/scaleway/scaleway-sdk-go/api/block/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/logger"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func SweepVolumes(scwClient *scw.Client, zone scw.Zone) error {
	blockAPI := block.NewAPI(scwClient)
	logger.Warningf("sweeper: destroying block volumes in %q", zone)

	listVolumes, err := blockAPI.ListVolumes(&block.ListVolumesRequest{
		Zone: zone,
	}, scw.WithAllPages())
	if err != nil {
		return fmt.Errorf("error listing block volumes: %w", err)
	}

	errs := []error(nil)
	terminalStatus := block.VolumeStatusAvailable
	for _, volume := range listVolumes.Volumes {
		_, err = blockAPI.WaitForVolume(&block.WaitForVolumeRequest{
			Zone:           zone,
			VolumeID:       volume.ID,
			TerminalStatus: &terminalStatus,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("error waiting for block volume %s: %w", volume.ID, err))
		}

		err = blockAPI.DeleteVolume(&block.DeleteVolumeRequest{
			VolumeID: volume.ID,
			Zone:     zone,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("error deleting block volume %s: %w", volume.ID, err))
		}
	}

	return errors.Join(errs...)
}

func SweepSnapshots(scwClient *scw.Client, zone scw.Zone) error {
	blockAPI := block.NewAPI(scwClient)
	logger.Warningf("sweeper: destroying block snapshots in %q", zone)

	listSnapshots, err := blockAPI.ListSnapshots(&block.ListSnapshotsRequest{
		Zone: zone,
	}, scw.WithAllPages())
	if err != nil {
		return fmt.Errorf("error listing block snapshots: %w", err)
	}

	errs := []error(nil)
	for _, snapshot := range listSnapshots.Snapshots {
		err := blockAPI.DeleteSnapshot(&block.DeleteSnapshotRequest{
			SnapshotID: snapshot.ID,
			Zone:       zone,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("error deleting snapshot %s: %w", snapshot.ID, err))
		}
	}

	return errors.Join(errs...)
}

func SweepAllLocalities(scwClient *scw.Client) error {
	errs := []error(nil)
	for _, zone := range (&block.API{}).Zones() {
		err := SweepVolumes(scwClient, zone)
		if err != nil {
			errs = append(errs, err)
		}
		err = SweepSnapshots(scwClient, zone)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
