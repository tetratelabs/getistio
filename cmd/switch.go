// Copyright 2021 Tetrate
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/tetratelabs/getistio/api"
	"github.com/tetratelabs/getistio/src/istioctl"
	"github.com/tetratelabs/getistio/src/util/logger"
)

func newSwitchCmd(homedir string) *cobra.Command {
	var (
		flagName          string
		flagVersion       string
		flagFlavor        string
		flagFlavorVersion int
	)

	cmd := &cobra.Command{
		Use:   "switch <istio version>",
		Short: "Switch the active istioctl to a specified version",
		Long:  `Switch the active istioctl to a specified version`,
		Example: `# switch the active istioctl version to version=1.7.4, flavor=tetrate and flavor-version=1
$ getistio switch --version 1.7.4 --flavor tetrate --flavor-version=1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := switchParse(homedir, flagName, flagVersion, flagFlavor, flagFlavorVersion)
			if err != nil {
				return err
			}
			return switchExec(homedir, d)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false
	flags.StringVarP(&flagName, "name", "", "", "Name of istioctl, e.g. 1.9.0-istio-v0")
	flags.StringVarP(&flagVersion, "version", "", "", "Version of istioctl, e.g. 1.7.4")
	flags.StringVarP(&flagFlavor, "flavor", "", "", "Flavor of istioctl, e.g. \"tetrate\" or \"tetratefips\" or \"istio\"")
	flags.IntVarP(&flagFlavorVersion, "flavor-version", "", -1, "Version of the flavor, e.g. 1")

	return cmd
}

// if set name, it should only parse name to distro
// if version, flavor and version are all set, just parse it to distro
// if there exists active distro, switch with only one or two command will use the active distro setting for unset command
// if there are no active distro exists, switch with only one or two command will use the default distro setting for unset command
// if all commands are not set, use active setting if there has otherwise use default version
// default version: latest version, default flavor: tetrate, default flavorversion: 0
func switchParse(homedir, flagName, flagVersion, flagFlavor string, flagFlavorVersion int) (*api.IstioDistribution, error) {
	if len(flagName) != 0 {
		d, err := api.IstioDistributionFromString(flagName)
		if err != nil {
			logger.Infof("cannot parse given name to %s istio distribution\n", flagName)
			return nil, err
		}
		return d, err
	}
	fetched, err := istioctl.GetFetchedVersions(homedir)
	if err != nil {
		logger.Infof("cannot fetch istio manifest\n")
		return nil, err
	}

	currDistro, err := istioctl.GetCurrentExecutable(homedir)
	if err != nil {
		return switchHandleDistro(nil, fetched[0].Version, flagVersion, flagFlavor, flagFlavorVersion)
	}
	return switchHandleDistro(currDistro, fetched[0].Version, flagVersion, flagFlavor, flagFlavorVersion)
}

func switchHandleDistro(curr *api.IstioDistribution, latestVersion, flagVersion, flagFlavor string,
	flagFlavorVersion int) (*api.IstioDistribution, error) {
	var defaultVersion, defaultFlavor string
	var defaultFlavorVersion int64

	if curr == nil {
		defaultVersion, defaultFlavor, defaultFlavorVersion = latestVersion, api.IstioDistributionFlavorTetrate, 0
	} else {
		defaultVersion, defaultFlavor, defaultFlavorVersion = curr.Version, curr.Flavor, curr.FlavorVersion
	}

	d := &api.IstioDistribution{
		Version:       defaultVersion,
		Flavor:        defaultFlavor,
		FlavorVersion: defaultFlavorVersion,
	}

	if len(flagVersion) != 0 {
		d.Version = flagVersion
	}
	if len(flagFlavor) != 0 {
		d.Flavor = flagFlavor
	}
	if flagFlavorVersion != -1 {
		d.FlavorVersion = int64(flagFlavorVersion)
	}
	return d, nil
}

func switchExec(homedir string, distribution *api.IstioDistribution) error {
	if err := istioctl.Switch(homedir, distribution); err != nil {
		return err
	}
	logger.Infof("istioctl switched to %s now\n", distribution.ToString())
	return nil
}
