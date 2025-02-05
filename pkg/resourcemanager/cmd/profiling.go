// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import "github.com/spf13/pflag"

// ProfilingOptions contains options needed to enable profiling.
type ProfilingOptions struct {
	// EnableProfiling enables profiling via web interface host:port/debug/pprof/.
	EnableProfiling bool
	// EnableContentionProfiling enable lock contention profiling, if profiling is enabled
	EnableContentionProfiling bool
}

// AddFlags adds the needed command line flags to the given FlagSet.
func (p *ProfilingOptions) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.EnableProfiling, "profiling", false, "Enable profiling via web interface host:port/debug/pprof/")
	fs.BoolVar(&p.EnableContentionProfiling, "contention-profiling", false, "Enable lock contention profiling, if profiling is enabled")
}
