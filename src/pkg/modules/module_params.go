// Package modules provides fucntionality to install and sign Linux kernel modules.
package modules

import (
	"fmt"

	"cos.googlesource.com/cos/tools.git/src/pkg/utils"
)

type ModuleParameters map[string][]string

func NewModuleParameters() ModuleParameters {
	return make(map[string][]string)
}

func (i *ModuleParameters) Set(value string) error {
	module, keyValue, found := utils.Cut(value, ".")
	if !found {
		return fmt.Errorf("modules: cannot parse module parameter %s, must be of form module.key=value", value)
	}
	moduleParamKey, moduleParamVal, found := utils.Cut(keyValue, "=")
	if !found || len(moduleParamKey) == 0 || len(moduleParamVal) == 0 {
		return fmt.Errorf("modules: cannot parse module parameter %s, must be of form module.key=value", value)
	}
	(*i)[module] = append((*i)[module], keyValue)
	return nil
}

func (i *ModuleParameters) String() string {
	return ""
}
