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

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/tetratelabs/getmesh/api"
	"github.com/tetratelabs/getmesh/src/getmesh"
	"github.com/tetratelabs/getmesh/src/istioctl"
	"github.com/tetratelabs/getmesh/src/util"
)

func TestMain(m *testing.M) {
	if err := os.Chdir(".."); err != nil {
		log.Fatal(err)
	}

	// set up manifest
	if err := os.Setenv("GETMESH_TEST_MANIFEST_PATH", "site/manifest.json"); err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func Test_E2E(t *testing.T) {
	t.Run("getmesh_install", getmeshInstall)
	t.Run("list", list)
	t.Run("end_of_life", enfOfLife)
	t.Run("security_patch_checker", securityPatchChecker)
	t.Run("fetch", fetch)
	t.Run("prune", prune)
	t.Run("show", show)
	t.Run("switch", switchTest)
	t.Run("istioctl_install", istioctlInstall)
	t.Run("unknown", unknown)
	t.Run("version", version)
	t.Run("check-upgrade", checkUpgrade)
	t.Run("config-validate", configValidate)
}

func getmeshInstall(t *testing.T) {
	cmd := exec.Command("bash", "site/install.sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())

	// check directory
	u, err := user.Current()
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(u.HomeDir, ".getmesh", "bin", "getmesh"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(u.HomeDir, ".getmesh", "istio"))
	require.NoError(t, err)
}

func securityPatchChecker(t *testing.T) {
	m := &api.Manifest{
		IstioDistributions: []*api.IstioDistribution{
			{
				Version:         "1.9.1000000000000",
				Flavor:          api.IstioDistributionFlavorTetrate,
				FlavorVersion:   0,
				IsSecurityPatch: true,
			},
		},
	}

	raw, err := json.Marshal(m)
	require.NoError(t, err)

	f, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer f.Close()

	_, err = f.Write(raw)
	require.NoError(t, err)

	cmd := exec.Command("./getmesh", "list")
	buf := new(bytes.Buffer)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("GETMESH_TEST_MANIFEST_PATH=%s", f.Name()))
	require.NoError(t, cmd.Run())
	require.Contains(t, buf.String(), `[WARNING] The locally installed minor version 1.9-tetrate has a latest version 1.9.1000000000000-tetrate-v0 including security patches. We strongly recommend you to download 1.9.1000000000000-tetrate-v0 by "getmesh fetch".`)
}

func enfOfLife(t *testing.T) {
	h, err := util.GetmeshHomeDir()
	require.NoError(t, err)
	require.NoError(t, getmesh.SetIstioVersion(h, &api.IstioDistribution{Version: "1.6.2"}))

	cmd := exec.Command("./getmesh", "list")
	buf := new(bytes.Buffer)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())
	require.Contains(t, buf.String(), `Your current active minor version 1.6 is reaching the end of life on 2020-11-21. We strongly recommend you to upgrade to the available higher minor versions`)
}

func list(t *testing.T) {
	cmd := exec.Command("./getmesh", "list")
	buf := new(bytes.Buffer)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())

	exp := `ISTIO VERSION	  FLAVOR   	FLAVOR VERSION	   K8S VERSIONS     
   *1.9.5    	  tetrate  	      0       	1.17,1.18,1.19,1.20	
    1.9.5    	   istio   	      0       	1.17,1.18,1.19,1.20	
    1.9.4    	  tetrate  	      0       	1.17,1.18,1.19,1.20	
    1.9.4    	   istio   	      0       	1.17,1.18,1.19,1.20	
    1.9.0    	  tetrate  	      0       	1.17,1.18,1.19,1.20	
    1.9.0    	tetratefips	      1       	1.17,1.18,1.19,1.20	
    1.9.0    	   istio   	      0       	1.17,1.18,1.19,1.20	
    1.8.6    	  tetrate  	      0       	1.16,1.17,1.18,1.19	
    1.8.6    	   istio   	      0       	1.16,1.17,1.18,1.19	
    1.8.5    	  tetrate  	      0       	1.16,1.17,1.18,1.19	
    1.8.5    	   istio   	      0       	1.16,1.17,1.18,1.19	
    1.8.3    	  tetrate  	      0       	1.16,1.17,1.18,1.19	
    1.8.3    	tetratefips	      1       	1.16,1.17,1.18,1.19	
    1.8.3    	   istio   	      0       	1.16,1.17,1.18,1.19	
    1.7.8    	  tetrate  	      0       	  1.16,1.17,1.18   	
    1.7.8    	   istio   	      0       	  1.16,1.17,1.18`
	require.Contains(t, buf.String(), exp)
}

func fetch(t *testing.T) {
	defer func() {
		cmd := exec.Command("./getmesh", "switch",
			"--version", "1.9.5", "--flavor", "tetrate", "--flavor-version=0",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())
	}()

	cmd := exec.Command("./getmesh", "fetch", "--version=1.8.6", "--flavor=tetrate", "--flavor-version=0")
	buf := new(bytes.Buffer)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run(), buf.String())
	require.Contains(t, buf.String(), `For more information about 1.8.6-tetrate-v0, please refer to the release notes: 
- https://istio.io/latest/news/releases/1.8.x/announcing-1.8.6/

istioctl switched to 1.8.6-tetrate-v0 now
`)

	// not listed version should be error
	cmd = exec.Command("./getmesh", "fetch", "--version=1.70000000000000000000.4")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.Error(t, cmd.Run())

	cmd = exec.Command("./getmesh", "fetch", "--version=1.70000000000000000000.4", "--flavor-version=0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.Error(t, cmd.Run())

	// fetch without version
	cmd = exec.Command("./getmesh", "fetch", "--flavor=istio", "--flavor-version=0")
	buf = new(bytes.Buffer)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())
	require.Contains(t, buf.String(), `-istio-v0 now`)

	// fetch with single flavor flag
	cmd = exec.Command("./getmesh", "fetch", "--flavor=istio")
	buf = new(bytes.Buffer)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())
	require.Contains(t, buf.String(), `-istio-v0 now`)

	// fetch another version
	cmd = exec.Command("./getmesh", "fetch", "--version=1.7.8")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())

	// check the active istioctl has been changed to the last fetched one
	cmd = exec.Command("./getmesh", "show")
	buf = new(bytes.Buffer)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())
	require.Contains(t, buf.String(), `1.7.8-tetrate-v0 (Active)`)
}

func prune(t *testing.T) {
	home, err := util.GetmeshHomeDir()
	require.NoError(t, err)

	// note that this prune test depends on the abovefetch test,
	// and we should restore the fetched versions for subsequent tests

	t.Run("specific", func(t *testing.T) {
		target := &api.IstioDistribution{
			Version:       "1.7.8",
			Flavor:        "tetrate",
			FlavorVersion: 0,
		}

		// should exist
		_, err = os.Stat(istioctl.GetIstioctlPath(home, target))
		require.NoError(t, err)

		// prune
		cmd := exec.Command("./getmesh", "prune", "--version", target.Version,
			"--flavor", target.Flavor, "--flavor-version", strconv.Itoa(int(target.FlavorVersion)))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())

		// should not exist
		_, err = os.Stat(istioctl.GetIstioctlPath(home, target))
		require.Error(t, err)

		// restore the version
		cmd = exec.Command("./getmesh", "fetch", "--version", target.Version,
			"--flavor", target.Flavor, "--flavor-version", strconv.Itoa(int(target.FlavorVersion)))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())
	})

	t.Run("all", func(t *testing.T) {
		distros := []*api.IstioDistribution{
			{
				Version:       "1.7.8",
				Flavor:        "tetrate",
				FlavorVersion: 0,
			},
			{
				Version:       "1.8.6",
				Flavor:        "tetrate",
				FlavorVersion: 0,
			},
			{
				Version:       "1.9.5",
				Flavor:        "tetrate",
				FlavorVersion: 0,
			},
		}
		for _, d := range distros {
			// should exist
			_, err = os.Stat(istioctl.GetIstioctlPath(home, d))
			require.NoError(t, err)
		}

		// prune all except the active one
		cmd := exec.Command("./getmesh", "prune")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())

		for i, d := range distros {
			if i == 0 {
				// should exist
				_, err = os.Stat(istioctl.GetIstioctlPath(home, d))
				require.NoError(t, err)
			} else {
				// should not exist
				_, err = os.Stat(istioctl.GetIstioctlPath(home, d))
				require.Error(t, err)

				// restore the version
				cmd = exec.Command("./getmesh", "fetch", "--version", d.Version,
					"--flavor", d.Flavor, "--flavor-version", strconv.Itoa(int(d.FlavorVersion)))
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				require.NoError(t, cmd.Run())
			}
		}
	})
}

func show(t *testing.T) {
	cmd := exec.Command("./getmesh", "show")
	buf := new(bytes.Buffer)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())
	exp := `1.7.8-tetrate-v0
1.8.6-tetrate-v0
1.9.5-tetrate-v0 (Active)`
	require.Contains(t, buf.String(), exp)
}

func switchTest(t *testing.T) {
	t.Run("full", func(t *testing.T) {
		for _, v := range []string{"1.8.6", "1.9.5"} {
			{
				cmd := exec.Command("./getmesh", "switch",
					"--version", v, "--flavor", "tetrate", "--flavor-version=0",
				)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				require.NoError(t, cmd.Run())
			}
			{
				cmd := exec.Command("./getmesh", "istioctl", "version")
				buf := new(bytes.Buffer)
				cmd.Stdout = buf
				cmd.Stderr = os.Stderr
				require.NoError(t, cmd.Run())
				require.Contains(t, buf.String(), v)
			}
		}
	})
	t.Run("name", func(t *testing.T) {
		cmd := exec.Command("./getmesh", "switch",
			"--version", "1.8.6", "--flavor", "tetrate", "--flavor-version=0",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())

		cmd = exec.Command("./getmesh", "istioctl", "version")
		buf := new(bytes.Buffer)
		cmd.Stdout = buf
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())
		require.Contains(t, buf.String(), "1.8.6-tetrate-v0")

		cmd = exec.Command("./getmesh", "switch",
			"--name", "1.9.5-tetrate-v0",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())

		cmd = exec.Command("./getmesh", "istioctl", "version")
		buf = new(bytes.Buffer)
		cmd.Stdout = buf
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())
		require.Contains(t, buf.String(), "1.9.5-tetrate-v0")
	})
	t.Run("active", func(t *testing.T) {
		cmd := exec.Command("./getmesh", "fetch",
			"--version=1.9.5", "--flavor=istio", "--flavor-version=0",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())

		cmd = exec.Command("./getmesh", "istioctl", "version")
		buf := new(bytes.Buffer)
		cmd.Stdout = buf
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())
		require.Contains(t, buf.String(), "1.9.5")
		require.NotContains(t, buf.String(), "1.9.5-tetrate-v0")

		cmd = exec.Command("./getmesh", "switch",
			"--flavor=tetrate",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())

		cmd = exec.Command("./getmesh", "istioctl", "version")
		buf = new(bytes.Buffer)
		cmd.Stdout = buf
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())
		require.Contains(t, buf.String(), "1.9.5-tetrate-v0")
	})
}

func istioctlInstall(t *testing.T) {
	cmd := exec.Command("./getmesh", "istioctl",
		"install", "--set", "profile=default", "-y")
	buf := new(bytes.Buffer)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())
	actual := buf.String()

	// istioctl x precheck
	require.Contains(t, actual, "Can initialize the Kubernetes client.")
	require.Contains(t, actual, "Can query the Kubernetes API Server.")
	require.Contains(t, actual, "Istio will be installed in the istio-system namespace.")
	require.Contains(t, actual, "Install Pre-Check passed! The cluster is ready for Istio installation.")
}

func unknown(t *testing.T) {
	cases := []struct {
		name  string
		cmd   *exec.Cmd
		wants string
	}{
		{
			name:  "unknown commands",
			cmd:   exec.Command("./getmesh", "unknown"),
			wants: `getmesh is an integration and lifecycle management CLI tool that ensures the use of supported and trusted versions of Istio.`,
		},
		{
			name:  "unknown flags",
			cmd:   exec.Command("./getmesh", "list", "--unknown"),
			wants: `List available Istio distributions built by Tetrate`,
		},
		{
			name:  "general tests",
			cmd:   exec.Command("./getmesh", "unknown", "list"),
			wants: `getmesh is an integration and lifecycle management CLI tool that ensures the use of supported and trusted versions of Istio.`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			c.cmd.Stdout = buf
			c.cmd.Stderr = os.Stderr
			require.Error(t, c.cmd.Run())
			actual := buf.String()
			require.Contains(t, actual, c.wants)
		})
	}
}

func version(t *testing.T) {
	t.Run("remote", func(t *testing.T) {
		for _, args := range [][]string{
			{"version", "--remote=true"},
			{"version"},
		} {
			cmd := exec.Command("./getmesh", args...)
			buf := new(bytes.Buffer)
			cmd.Stdout = buf
			cmd.Stderr = os.Stderr
			require.NoError(t, cmd.Run())
			actual := buf.String()
			require.Contains(t, actual, "getmesh version: dev")
			require.Contains(t, actual, "active istioctl")
			// latest version is available
			require.Contains(t, actual, "control plane version")
			require.Contains(t, actual, "data plane version")
			fmt.Println(actual)
		}

	})
	t.Run("local", func(t *testing.T) {
		cmd := exec.Command("./getmesh", "version", "--remote=false")
		buf := new(bytes.Buffer)
		cmd.Stdout = buf
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())
		actual := buf.String()
		require.Contains(t, actual, "active istioctl")
		// latest version is available
		require.NotContains(t, actual, "control plane version")
		require.NotContains(t, actual, "data plane version")
	})
	t.Run("unknown cluster", func(t *testing.T) {
		cmd := exec.Command("./getmesh", "version", "-c", "unknown.yaml")
		buf := new(bytes.Buffer)
		cmd.Stdout = buf
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())
		actual := buf.String()
		require.Contains(t, actual, "active istioctl")
		require.Contains(t, actual, "no active Kubernetes clusters found")
	})
}

func checkUpgrade(t *testing.T) {
	cmd := exec.Command("./getmesh", "check-upgrade")
	buf := new(bytes.Buffer)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run(), buf.String())
	actual := buf.String()
	require.Contains(t, actual, "1.9.5-tetrate-v0 is the latest version in 1.9-tetrate")

	// change image to 1.8.1-tetrate-v0
	image := "containers.istio.tetratelabs.com/pilot:1.8.1-tetrate-v0"
	patch := fmt.Sprintf(`{"spec":{"template":{"spec":{"containers":[{"name":"discovery","image":"%s"}]}}}}`,
		image)
	cmd = exec.Command("kubectl", "patch", "deployment",
		"-nistio-system", "istiod", "-p", patch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())

	var i int
	for ; i < 10; i++ {
		time.Sleep(time.Second * 6)
		cmd := exec.Command("./getmesh", "check-upgrade")
		buf := new(bytes.Buffer)
		cmd.Stdout = buf
		cmd.Stderr = os.Stderr
		_ = cmd.Run()

		actual := buf.String()
		// - There is the available patch for the minor version 1.8-tetrate which includes **security upgrades**. We strongly recommend upgrading all 1.8-tetrate versions -> 1.8.6-tetrate-v0
		if strings.Contains(actual,
			"There is the available patch for the minor version 1.8-tetrate which includes **security upgrades**. "+
				"We strongly recommend upgrading all 1.8-tetrate versions -> 1.8.6-tetrate-v0") {
			break
		}
	}

	require.NotEqual(t, 10, i)
}

func configValidate(t *testing.T) {
	cmd := exec.Command("kubectl", "apply", "-f", "./e2e/testdata/config-validate.yaml")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())
	time.Sleep(time.Second * 6)

	t.Run("all namespaces", func(t *testing.T) {
		t.Parallel()
		cmd := exec.Command("./getmesh", "config-validate")
		bufOut := new(bytes.Buffer)
		cmd.Stdout = bufOut
		cmd.Stderr = os.Stderr
		require.Error(t, cmd.Run())
		exps := []string{
			`IST0101`, `Referenced selector not found: "app=nonexisting"`,
			`KIA0505`, `Destination Rule disabling namespace-wide mTLS is missing`,
			`KIA1102`, `VirtualService is pointing to a non-existent gateway`,
		}

		out := bufOut.String()
		for _, exp := range exps {
			require.Contains(t, out, exp, exp)
		}
		fmt.Println(out)
	})

	t.Run("all namespaces with threshold", func(t *testing.T) {
		t.Parallel()
		cmd := exec.Command("./getmesh", "config-validate", "--output-threshold", "Error")
		bufOut := new(bytes.Buffer)
		cmd.Stdout = bufOut
		cmd.Stderr = os.Stderr
		require.Error(t, cmd.Run())

		out := bufOut.String()
		for _, exp := range []string{"Info", "Warning"} {
			require.NotContains(t, out, exp, exp)
		}
		fmt.Println(out)
	})

	t.Run("invalid kubeconfig", func(t *testing.T) {
		t.Parallel()
		// make a new location for config
		// TODO: misconfigured kubeconfig, i,e: unauthorized kubeconfig file
		f, err := ioutil.TempFile("", "")
		require.NoError(t, err)
		defer f.Close()
		cmd := exec.Command("./getmesh", "config-validate", "--kubeconfig", f.Name())
		bufErr := new(bytes.Buffer)
		cmd.Stderr = bufErr
		require.Error(t, cmd.Run())

		out := bufErr.String()
		exp := fmt.Sprintf("error building config from kubeconfig located in %s", f.Name())
		require.Contains(t, out, exp)
	})

	t.Run("single namespace", func(t *testing.T) {
		t.Parallel()
		cmd := exec.Command("./getmesh", "config-validate", "-n", "bookinfo")
		bufOut := new(bytes.Buffer)
		cmd.Stdout = bufOut
		cmd.Stderr = os.Stderr
		require.Error(t, cmd.Run())

		exps := []string{
			`IST0101`, `Referenced selector not found: "app=nonexisting"`,
			`KIA0505`, `Destination Rule disabling namespace-wide mTLS is missing`,
			`KIA1102`, `VirtualService is pointing to a non-existent gateway`,
		}
		out := bufOut.String()
		for _, exp := range exps {
			require.Contains(t, out, exp, exp)
		}
		fmt.Println(out)
	})

	t.Run("healthy", func(t *testing.T) {
		t.Parallel()
		cmd := exec.Command("./getmesh", "config-validate", "-n", "healthy")
		bufOut := new(bytes.Buffer)
		cmd.Stdout = bufOut
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())

		out := bufOut.String()
		exp := "Configuration issues not found."
		require.Contains(t, out, exp, exp)
		fmt.Println(out)
	})

	t.Run("local file", func(t *testing.T) {
		t.Parallel()
		cmd := exec.Command("./getmesh",
			"config-validate", "-n", "invalid",
			"e2e/testdata/config-validate-local.yaml",
		)
		bufOut := new(bytes.Buffer)
		cmd.Stdout = bufOut
		cmd.Stderr = os.Stderr
		require.Error(t, cmd.Run())

		exps := []string{
			`IST0101`, `ratings-bogus-weight-default`,
			`[e2e/testdata/config-validate-local.yaml:29] Referenced host+subset in destinationrule not found: "ratings+v1`,
			`KIA1104`, `[e2e/testdata/config-validate-local.yaml] The weight is assumed to be 100 because there is only one route destination`,
		}
		out := bufOut.String()
		for _, exp := range exps {
			require.Contains(t, out, exp, exp)
		}
		fmt.Println(out)
	})

	t.Run("local directory", func(t *testing.T) {
		t.Parallel()
		cmd := exec.Command("./getmesh",
			"config-validate", "-n", "invalid",
			"e2e/testdata/config-validate-local",
		)
		bufOut := new(bytes.Buffer)
		cmd.Stdout = bufOut
		cmd.Stderr = os.Stderr
		require.Error(t, cmd.Run())

		exps := []string{
			`IST0108`,
			`[e2e/testdata/config-validate-local/config-validate-local.yaml:1] Unknown annotation: networking.istio.io/non-exist`,
		}
		out := bufOut.String()
		for _, exp := range exps {
			require.Contains(t, out, exp, exp)
		}
		fmt.Println(out)
	})
}
