package cli

import (
	"context"
	"reflect"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/cirun"
	"github.com/spf13/cobra"
)

func TestCIHistoryRoutesLegacyProfileAndExplicitAnalysis(t *testing.T) {
	for _, tc := range []struct {
		name                            string
		args                            []string
		wantParent, wantStatus, wantErr bool
		wantFresh, wantLookup, wantDry  bool
	}{
		{name: "bare profile", args: []string{"history"}, wantParent: true},
		{name: "fresh after profile", args: []string{"history", "--fresh"}, wantParent: true, wantFresh: true},
		{name: "fresh before profile", args: []string{"--fresh", "history"}, wantParent: true, wantFresh: true},
		{name: "lookup", args: []string{"history", "--lookup"}, wantParent: true, wantLookup: true},
		{name: "dry run", args: []string{"history", "--dry-run"}, wantParent: true, wantDry: true},
		{name: "root only", args: []string{"history", "--root", "/tmp"}, wantErr: true},
		{name: "profile only", args: []string{"history", "--profile", "full"}, wantErr: true},
		{name: "empty root", args: []string{"history", "--root=", "--profile", "full"}, wantErr: true},
		{name: "empty profile", args: []string{"history", "--root", "/tmp", "--profile= "}, wantErr: true},
		{name: "explicit analysis", args: []string{"history", "--root", "/tmp", "--profile", "full"}, wantStatus: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			type contextKey struct{}
			ctx := context.WithValue(context.Background(), contextKey{}, "marker")
			var parentCalls, statusCalls int
			var gotArgs []string
			var fresh, lookup, dry bool
			parent := &cobra.Command{Use: "ci [profile]", SilenceErrors: true, SilenceUsage: true,
				RunE: func(cmd *cobra.Command, args []string) error {
					parentCalls++
					gotArgs = args
					if cmd.Context().Value(contextKey{}) != "marker" {
						t.Error("lost command context")
					}
					return nil
				}}
			parent.Flags().BoolVar(&fresh, "fresh", false, "")
			parent.Flags().BoolVar(&lookup, "lookup", false, "")
			parent.PersistentFlags().BoolVar(&dry, "dry-run", false, "")
			parent.AddCommand(newCIHistoryCommand(parent, func(string) ([]cirun.Report, error) {
				statusCalls++
				return nil, nil
			}))
			parent.SetArgs(tc.args)
			err := parent.ExecuteContext(ctx)
			if (err != nil) != tc.wantErr || (parentCalls == 1) != tc.wantParent || (statusCalls == 1) != tc.wantStatus {
				t.Fatalf("err=%v parent=%d status=%d", err, parentCalls, statusCalls)
			}
			if tc.wantParent && !reflect.DeepEqual(gotArgs, []string{"history"}) {
				t.Fatalf("args=%v", gotArgs)
			}
			if fresh != tc.wantFresh || lookup != tc.wantLookup || dry != tc.wantDry {
				t.Fatalf("flags fresh=%v lookup=%v dry=%v", fresh, lookup, dry)
			}
		})
	}
}
