package launcher

import (
	"path/filepath"
	"strings"
)

const (
	configurationCacheEnabledArgument   = "--configuration-cache"
	configurationCacheDisabledArgument  = "--no-configuration-cache"
	configurationCacheProblemsPrefix    = "--configuration-cache-problems="
	configurationPolicyPropertyPrefix   = "-Dbuildopt.configuration-policy.digest="
	configurationContractPropertyPrefix = "-Dbuildopt.configuration-contract.version="
)

func applyConfigurationCachePolicy(childArgs []string, authority *localAuthorityContext) []string {
	if authority == nil || authority.configurationCacheContract == "" || len(childArgs) == 0 || filepath.Base(childArgs[0]) != "gradlew" {
		return childArgs
	}
	result := make([]string, 0, len(childArgs)+4)
	result = append(result, childArgs[0])
	if authority.configurationCacheEnabled {
		result = append(result, configurationCacheEnabledArgument, configurationCacheProblemsPrefix+"fail")
	} else {
		result = append(result, configurationCacheDisabledArgument)
	}
	result = append(result,
		configurationPolicyPropertyPrefix+authority.configurationPolicyDigest,
		configurationContractPropertyPrefix+authority.configurationCacheContract,
	)
	for _, argument := range childArgs[1:] {
		if argument == configurationCacheEnabledArgument || argument == configurationCacheDisabledArgument ||
			strings.HasPrefix(argument, configurationCacheProblemsPrefix) ||
			strings.HasPrefix(argument, configurationPolicyPropertyPrefix) ||
			strings.HasPrefix(argument, configurationContractPropertyPrefix) {
			continue
		}
		result = append(result, argument)
	}
	return result
}
