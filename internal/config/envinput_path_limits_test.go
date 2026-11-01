package config

import (
	"reflect"
	"testing"
)

func TestUnresolvedPathRefsNamesTheCause(t *testing.T) {
	env := NewEnvironment(map[string]string{
		"DVA_T_SET":   "x",
		"DVA_T_EMPTY": "",
		"DVA_T_REF":   "${DVA_T_OTHER}",
		"DVA_T_BAD":   "${DVA_T_OTHER:?x}",
		"DVA_T_DOLL":  "a$",
	}, t.TempDir(), t.TempDir())

	for _, tt := range []struct {
		path string
		want []string
	}{
		{path: "lit${DVA_T_REF}", want: []string{"${DVA_T_OTHER}"}},
		{path: "${DVA_T_BAD}", want: []string{"${DVA_T_OTHER:?x}"}},
		{path: "${DVA_T_A:?x}${DVA_T_SET}", want: []string{"${DVA_T_A:?x}"}},
		{path: "${DVA_T_UNSET_A:-}/", want: []string{"${DVA_T_UNSET_A:-}/"}},
		{path: "./${DVA_T_EMPTY}", want: []string{"./${DVA_T_EMPTY}"}},
		{path: "${DVA_T_SET_UNDEF:-x}${DVA_T_A:?x}", want: []string{"${DVA_T_A:?x}"}},
		{path: "${DVA_T_EMPTY}/.env", want: []string{"${DVA_T_EMPTY}/.env"}},
		{path: "$DVA_T_EMPTY/.env", want: []string{"$DVA_T_EMPTY/.env"}},
		{path: "${DVA_T_EMPTY}${DVA_T_UNSET_A:-}/.env", want: []string{"${DVA_T_EMPTY}${DVA_T_UNSET_A:-}/.env"}},
		{path: "${DVA_T_UNSET_A:-${DVA_T_EMPTY}/.env}", want: []string{"${DVA_T_UNSET_A:-${DVA_T_EMPTY}/.env}"}},
		{path: "${DVA_T_UNSET_A-$DVA_T_EMPTY/.env}", want: []string{"${DVA_T_UNSET_A-$DVA_T_EMPTY/.env}"}},
		{path: "${DVA_T_UNSET_A:-${DVA_T_UNSET_B:-${DVA_T_EMPTY}/.env}}", want: []string{"${DVA_T_UNSET_A:-${DVA_T_UNSET_B:-${DVA_T_EMPTY}/.env}}"}},
		{path: "${DVA_T_EMPTY}${DVA_T_UNSET_A:-${DVA_T_EMPTY}/.env}", want: []string{"${DVA_T_EMPTY}${DVA_T_UNSET_A:-${DVA_T_EMPTY}/.env}"}},
		{path: "${DVA_T_UNSET_A:-${DVA_T_EMPTY}/}.env", want: []string{"${DVA_T_UNSET_A:-${DVA_T_EMPTY}/}.env"}},
		{path: "${DVA_T_EMPTY-${DVA_T_EMPTY}/x}.env"},
		{path: "${DVA_T_SET:-${DVA_T_EMPTY}/.env}"},
		{path: "${DVA_T_EMPTY}$/.env"},
		{path: "${DVA_T_SET}/.env"},
		{path: "${DVA_T_EMPTY}.env"},
		{path: "${DVA_T_UNSET_A:-/abs}/.env"},
		{path: "/"},
		{path: ".env.${DVA_T_DOLL}"},
		{path: "/abs/.env"},
		{path: ".env"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			if got := unresolvedPathRefs(tt.path, env); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
