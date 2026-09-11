package cli

import (
	"sort"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/lifecycle"
)

type ManifestPlan struct {
	Description     string              `json:"description,omitempty" yaml:"description,omitempty"`
	Environment     string              `json:"environment,omitempty" yaml:"environment,omitempty"`
	Site            string              `json:"site,omitempty" yaml:"site,omitempty"`
	EndpointTags    []string            `json:"endpoint_tags,omitempty" yaml:"endpoint_tags,omitempty"`
	Entries         []ManifestPlanEntry `json:"entries" yaml:"entries"`
	ResolutionError string              `json:"resolution_error,omitempty" yaml:"resolution_error,omitempty"`
	Owner           string              `json:"owner" yaml:"owner"`
	Aliases         []string            `json:"aliases,omitempty" yaml:"aliases,omitempty"`
	AliasOf         string              `json:"alias_of,omitempty" yaml:"alias_of,omitempty"`
	Alias           string              `json:"alias,omitempty" yaml:"alias,omitempty"`
	Extends         string              `json:"extends,omitempty" yaml:"extends,omitempty"`
}

type ManifestPlanEntry struct {
	Name      string   `json:"name" yaml:"name"`
	Runner    string   `json:"runner" yaml:"runner"`
	Order     int      `json:"order" yaml:"order"`
	DependsOn []string `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Profiles  []string `json:"profiles,omitempty" yaml:"profiles,omitempty"`
	Services  []string `json:"services,omitempty" yaml:"services,omitempty"`
	Wave      int      `json:"wave" yaml:"wave"`
}

func planAliasGroups(plans map[string]*config.PlanConfig) map[string][]string {
	groups := make(map[string][]string)
	for k, p := range plans {
		if p == nil || p.CanonicalAddress == "" || k == p.CanonicalAddress {
			continue
		}
		groups[p.CanonicalAddress] = append(groups[p.CanonicalAddress], k)
	}
	for canonical := range groups {
		sort.Strings(groups[canonical])
	}
	return groups
}

func planOwnerName(p *config.PlanConfig) string {
	if p != nil && p.SubprojectName != "" {
		return p.SubprojectName
	}
	return rootOwnerName
}

func buildManifestPlans(c *config.Config) map[string]ManifestPlan {
	if len(c.Plans) == 0 {
		return nil
	}

	aliasGroups := planAliasGroups(c.Plans)

	plans := make(map[string]ManifestPlan, len(c.Plans))
	for _, name := range sortedKeys(c.Plans) {
		planConfig := c.Plans[name]
		if planConfig == nil {
			plans[name] = ManifestPlan{
				Entries:         []ManifestPlanEntry{},
				ResolutionError: "plan configuration is empty",
				Owner:           rootOwnerName,
			}
			continue
		}

		plan := ManifestPlan{
			Description:  planConfig.Description,
			Environment:  planConfig.Environment,
			Site:         planConfig.Site,
			EndpointTags: planConfig.EndpointTags,
			Entries:      make([]ManifestPlanEntry, 0, len(planConfig.Entries)),
			Owner:        planOwnerName(planConfig),
			Alias:        planConfig.Alias,
			Extends:      planConfig.Extends,
		}
		if planConfig.CanonicalAddress != "" {
			if name == planConfig.CanonicalAddress {
				if aliases := aliasGroups[name]; len(aliases) > 0 {
					plan.Aliases = aliases
				}
			} else {
				plan.AliasOf = planConfig.CanonicalAddress
			}
		}
		resolved, err := lifecycle.ResolvePlan(c, name, nil)
		if err != nil {
			plan.ResolutionError = err.Error()
			plans[name] = plan
			continue
		}

		for _, entry := range resolved.Entries {
			plan.Entries = append(plan.Entries, ManifestPlanEntry{
				Name:      entry.Name,
				Runner:    entry.Runner,
				Order:     entry.Order,
				DependsOn: entry.DependsOn,
				Profiles:  entry.Profiles,
				Services:  entry.Services,
				Wave:      entry.Wave,
			})
		}
		plans[name] = plan
	}
	return plans
}
