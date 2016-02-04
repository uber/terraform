package google

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	computeAlpha "google.golang.org/api/compute/v0.alpha"
)

// OperationWaitType is an enum specifying what type of operation
// we're waiting on.
type ComputeAlphaOperationWaitType byte

const (
	ComputeAlphaOperationWaitInvalid ComputeAlphaOperationWaitType = iota
	ComputeAlphaOperationWaitGlobal
	ComputeAlphaOperationWaitRegion
	ComputeAlphaOperationWaitZone
)

type ComputeAlphaOperationWaiter struct {
	Service *computeAlpha.Service
	Op      *computeAlpha.Operation
	Project string
	Region  string
	Type    ComputeAlphaOperationWaitType
	Zone    string
}

func (w *ComputeAlphaOperationWaiter) RefreshFunc() resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		var op *computeAlpha.Operation
		var err error

		switch w.Type {
		case ComputeAlphaOperationWaitGlobal:
			op, err = w.Service.GlobalOperations.Get(
				w.Project, w.Op.Name).Do()
		case ComputeAlphaOperationWaitRegion:
			op, err = w.Service.RegionOperations.Get(
				w.Project, w.Region, w.Op.Name).Do()
		case ComputeAlphaOperationWaitZone:
			op, err = w.Service.ZoneOperations.Get(
				w.Project, w.Zone, w.Op.Name).Do()
		default:
			return nil, "bad-type", fmt.Errorf(
				"Invalid wait type: %#v", w.Type)
		}

		if err != nil {
			return nil, "", err
		}

		log.Printf("[DEBUG] Got %q when asking for operation %q", op.Status, w.Op.Name)

		return op, op.Status, nil
	}
}

func (w *ComputeAlphaOperationWaiter) Conf() *resource.StateChangeConf {
	return &resource.StateChangeConf{
		Pending: []string{"PENDING", "RUNNING"},
		Target:  []string{"DONE"},
		Refresh: w.RefreshFunc(),
	}
}

// ComputeAlphaOperationError wraps computeAlpha.OperationError and implements the
// error interface so it can be returned.
type ComputeAlphaOperationError computeAlpha.OperationError

func (e ComputeAlphaOperationError) Error() string {
	var buf bytes.Buffer

	for _, err := range e.Errors {
		buf.WriteString(err.Message + "\n")
	}

	return buf.String()
}

func computeAlphaOperationWaitGlobal(config *Config, op *computeAlpha.Operation, activity string) error {
	w := &ComputeAlphaOperationWaiter{
		Service: config.clientComputeAlpha,
		Op:      op,
		Project: config.Project,
		Type:    ComputeAlphaOperationWaitGlobal,
	}

	state := w.Conf()
	state.Delay = 10 * time.Second
	state.Timeout = 4 * time.Minute
	state.MinTimeout = 2 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for %s: %s", activity, err)
	}

	op = opRaw.(*computeAlpha.Operation)
	if op.Error != nil {
		return ComputeAlphaOperationError(*op.Error)
	}

	return nil
}

func computeAlphaOperationWaitRegion(config *Config, op *computeAlpha.Operation, region, activity string) error {
	w := &ComputeAlphaOperationWaiter{
		Service: config.clientComputeAlpha,
		Op:      op,
		Project: config.Project,
		Type:    ComputeAlphaOperationWaitRegion,
		Region:  region,
	}

	state := w.Conf()
	state.Delay = 10 * time.Second
	state.Timeout = 4 * time.Minute
	state.MinTimeout = 2 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for %s: %s", activity, err)
	}

	op = opRaw.(*computeAlpha.Operation)
	if op.Error != nil {
		return ComputeAlphaOperationError(*op.Error)
	}

	return nil
}

func computeAlphaOperationWaitZone(config *Config, op *computeAlpha.Operation, zone, activity string) error {
	return computeAlphaOperationWaitZoneTime(config, op, zone, 4, activity)
}

func computeAlphaOperationWaitZoneTime(config *Config, op *computeAlpha.Operation, zone string, minutes int, activity string) error {
	w := &ComputeAlphaOperationWaiter{
		Service: config.clientComputeAlpha,
		Op:      op,
		Project: config.Project,
		Zone:    zone,
		Type:    ComputeAlphaOperationWaitZone,
	}
	state := w.Conf()
	state.Delay = 10 * time.Second
	state.Timeout = time.Duration(minutes) * time.Minute
	state.MinTimeout = 2 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for %s: %s", activity, err)
	}
	op = opRaw.(*computeAlpha.Operation)
	if op.Error != nil {
		// Return the error
		return ComputeAlphaOperationError(*op.Error)
	}
	return nil
}
