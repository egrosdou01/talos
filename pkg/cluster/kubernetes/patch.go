// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package kubernetes

import (
	"context"
	"fmt"

	"github.com/cosi-project/runtime/pkg/safe"

	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/config/container"
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	v1alpha1config "github.com/siderolabs/talos/pkg/machinery/config/types/v1alpha1"
	"github.com/siderolabs/talos/pkg/machinery/resources/config"
)

// patchNodeConfig updates node configuration by means of patch function.
func patchNodeConfig(ctx context.Context, cluster UpgradeProvider, node string, encoderOpt encoder.Option, patchFunc func(config *v1alpha1config.Config) error) error {
	c, err := cluster.Client()
	if err != nil {
		return fmt.Errorf("error building Talos API client: %w", err)
	}

	ctx = client.WithNode(ctx, node)

	mc, err := safe.StateGetByID[*config.MachineConfig](ctx, c.COSI, config.V1Alpha1ID)
	if err != nil {
		return fmt.Errorf("error fetching config resource: %w", err)
	}

	cfg := mc.Container().RawV1Alpha1()
	if cfg == nil {
		return fmt.Errorf("config is not v1alpha1 config")
	}

	if !cfg.Persist() {
		return fmt.Errorf("config persistence is disabled, patching is not supported")
	}

	if err = patchFunc(cfg); err != nil {
		return fmt.Errorf("error patching config: %w", err)
	}

	cfgBytes, err := container.NewV1Alpha1(cfg).EncodeBytes(encoderOpt)
	if err != nil {
		return fmt.Errorf("error serializing config: %w", err)
	}

	_, err = c.ApplyConfiguration(ctx, &machine.ApplyConfigurationRequest{
		Data: cfgBytes,
		Mode: machine.ApplyConfigurationRequest_NO_REBOOT,
	})
	if err != nil {
		return fmt.Errorf("error applying config: %w", err)
	}

	return nil
}
