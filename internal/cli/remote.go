package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/jobrun"
	"github.com/ScriptonBasestar/dva/internal/output"
	"github.com/ScriptonBasestar/dva/internal/remotetarget"
	"github.com/ScriptonBasestar/dva/internal/secretpush"
	"github.com/spf13/cobra"
)

var secretCmd = &cobra.Command{
	Use: "secret", Short: "Transfer declared secrets without printing plaintext", GroupID: "integration",
	Long: "Transfer explicitly selected keys from declared SOPS dotenv sources to this checkout's GitHub Actions secrets. Use 'dva secret push <target> --dry-run' to inspect the destination and key names before sending values.",
}
var jobCmd = &cobra.Command{
	Use: "job", Short: "Run and verify repository-owned artifact jobs", GroupID: "integration",
	Long: "Run named jobs from dva.yml, record exact GitHub workflow run IDs, and verify their published OCI images. Jobs own one-off remote artifact work; their local receipts support status, resume, and verify without dispatching again.",
}

// Seams exercise command routing and effect ordering without real remote writes.
var pushRemoteSecret = secretpush.Push
var runRemoteJob = jobrun.Run

func init() {
	push := &cobra.Command{Use: "push <target>", Short: "Push selected SOPS keys to this repository's GitHub Actions secrets", Args: cobra.ExactArgs(1), RunE: runSecretPush}
	push.Long = "Decrypt the target's SOPS dotenv source, validate all selected keys, and send each value to GitHub through stdin. The target repository must match the owning checkout's origin. Output contains key states only; partial failures stop later writes. Use --project to select a child declaration or --dry-run to preview without decrypting or writing."
	push.Flags().String("project", "", "Use a declared child project's own secret targets")
	secretCmd.AddCommand(push)
	run := &cobra.Command{Use: "run <name>", Short: "Dispatch a declared batch of GitHub Actions workflows", Args: cobra.ExactArgs(1), RunE: runJob}
	run.Long = "Resolve a named job and its public --input NAME=VALUE arguments, then dispatch its workflows and record exact run IDs. --wait waits for completion; --verify also compares workflow result artifacts with OCI digests and platforms. Secret targets are sent only with --with-secrets. --dry-run previews without remote requests or receipt writes."
	run.Flags().String("project", "", "Use a declared child project's own jobs")
	run.Flags().StringArray("input", nil, "Public job input NAME=VALUE (repeatable; never pass secrets)")
	run.Flags().Bool("wait", false, "Wait for the exact dispatched runs to finish")
	run.Flags().Bool("verify", false, "Wait and compare run artifact digests with the public OCI registry")
	run.Flags().Bool("with-secrets", false, "Push this job's declared secret_targets before dispatch")
	jobCmd.AddCommand(run)
	for _, verb := range []string{"status", "resume", "verify"} {
		child := &cobra.Command{Use: verb + " <run-id>", Args: cobra.ExactArgs(1)}
		switch verb {
		case "status":
			child.Short = "Refresh the exact remote runs in an existing local receipt"
			child.Long = "Query each recorded GitHub run once and update the local receipt. The run-id is the local job receipt ID returned by job run. This command does not reload configuration or dispatch new workflows."
		case "resume":
			child.Short = "Resume waiting for recorded runs without dispatching again"
			child.Long = "Wait for the exact GitHub runs in a local job receipt using its recorded timeout. --verify additionally validates declared OCI images after completion. An unconfirmed dispatch cannot be resumed automatically; no workflow is sent again."
			child.Flags().Bool("verify", false, "Verify result digests after the recorded runs succeed")
		case "verify":
			child.Short = "Verify recorded workflow artifacts against public OCI images"
			child.Long = "Refresh a local job receipt and verify its completed successful runs. Each image-producing run must provide the declared result artifact with matching head SHA, image references, and published digests. Compare those digests and requested platforms with the public registry; incomplete or mismatched evidence fails."
		}
		child.RunE = func(cmd *cobra.Command, args []string) error {
			if dryRun {
				return fmt.Errorf("--dry-run is supported by secret push and job run only")
			}
			ctx, stop := remoteContext(cmd)
			defer stop()
			var report jobrun.Report
			var err error
			switch cmd.Name() {
			case "status":
				report, err = jobrun.Status(ctx, "", args[0])
			case "resume":
				verify, _ := cmd.Flags().GetBool("verify")
				report, err = jobrun.Resume(ctx, "", args[0], verify)
			case "verify":
				report, err = jobrun.Verify(ctx, "", args[0])
			}
			return printRemoteResult(report, err)
		}
		jobCmd.AddCommand(child)
	}
	rootCmd.AddCommand(secretCmd, jobCmd)
}

func remoteContext(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
}

func remoteOwner(cmd *cobra.Command) (*config.Config, error) {
	c, err := loadConfig()
	if err != nil {
		return nil, err
	}
	project, _ := cmd.Flags().GetString("project")
	if project == "" {
		return c, c.ValidateRemoteDeclarations()
	}
	sub, ok := c.Subprojects[project]
	if !ok {
		return nil, fmt.Errorf("unknown project %q", project)
	}
	children, err := config.LoadSubprojects(c.FileDir(), map[string]config.SubprojectConfig{project: sub})
	if err != nil {
		return nil, err
	}
	child := children[project]
	return child, child.ValidateRemoteDeclarations()
}

func secretOptions(c *config.Config, name string) (secretpush.Options, error) {
	if c.Secrets == nil {
		return secretpush.Options{}, fmt.Errorf("no secret targets are declared")
	}
	target, ok := c.Secrets.Targets[name]
	if !ok {
		return secretpush.Options{}, fmt.Errorf("unknown secret target %q", name)
	}
	source, ok := c.Secrets.Sources[target.Source]
	if !ok {
		return secretpush.Options{}, fmt.Errorf("secret target source is undefined")
	}
	return secretpush.Options{Root: c.FileDir(), Name: name, DryRun: dryRun, Target: secretpush.Target{Source: source.Sops, Repository: target.Repository, Keys: target.Keys}}, nil
}

func runSecretPush(cmd *cobra.Command, args []string) error {
	c, err := remoteOwner(cmd)
	if err != nil {
		return err
	}
	opts, err := secretOptions(c, args[0])
	if err != nil {
		return err
	}
	ctx, stop := remoteContext(cmd)
	defer stop()
	report, err := pushRemoteSecret(ctx, opts)
	return printRemoteResult(report, err)
}

func runJob(cmd *cobra.Command, args []string) error {
	c, err := remoteOwner(cmd)
	if err != nil {
		return err
	}
	definition, ok := c.Jobs[args[0]]
	if !ok {
		return fmt.Errorf("unknown job %q", args[0])
	}
	inputArgs, _ := cmd.Flags().GetStringArray("input")
	inputs := map[string]string{}
	for _, input := range inputArgs {
		key, value, ok := strings.Cut(input, "=")
		if !ok || key == "" {
			return fmt.Errorf("job input must be NAME=VALUE")
		}
		if _, duplicate := inputs[key]; duplicate {
			return fmt.Errorf("duplicate job input %q", key)
		}
		inputs[key] = value
	}
	wait, _ := cmd.Flags().GetBool("wait")
	verify, _ := cmd.Flags().GetBool("verify")
	withSecrets, _ := cmd.Flags().GetBool("with-secrets")
	if withSecrets && len(definition.SecretTargets) == 0 {
		return fmt.Errorf("job declares no secret_targets")
	}
	ctx, stop := remoteContext(cmd)
	defer stop()
	opts := jobrun.Options{Root: c.FileDir(), Name: args[0], Definition: jobDefinition(definition), Inputs: inputs, Wait: wait, Verify: verify}
	resolved, err := jobrun.Resolve(opts.Definition, inputs)
	if err != nil {
		return err
	}
	if verify {
		hasImages := false
		for _, run := range resolved.Runs {
			hasImages = hasImages || len(run.Images) > 0
		}
		if !hasImages {
			return fmt.Errorf("job verification requires at least one declared image")
		}
	}
	if dryRun {
		if err := remotetarget.Validate(ctx, c.FileDir(), definition.Repository); err != nil {
			return err
		}
		return printRemoteResult(struct {
			Plan        jobrun.Definition `json:"plan"`
			PushSecrets bool              `json:"push_secrets"`
			Verify      bool              `json:"verify"`
		}{resolved, withSecrets, verify}, nil)
	}
	var secretReports []secretpush.Report
	if withSecrets {
		var targets []secretpush.Options
		for _, name := range definition.SecretTargets {
			target, err := secretOptions(c, name)
			if err != nil {
				return err
			}
			targets = append(targets, target)
		}
		opts.BeforeDispatch = func(ctx context.Context) error {
			for _, target := range targets {
				report, err := pushRemoteSecret(ctx, target)
				secretReports = append(secretReports, report)
				if err != nil {
					return err
				}
			}
			return nil
		}
	}
	report, err := runRemoteJob(ctx, opts)
	return printRemoteResult(struct {
		Job     jobrun.Report       `json:"job"`
		Secrets []secretpush.Report `json:"secrets,omitempty"`
	}{report, secretReports}, err)
}

func jobDefinition(c config.JobConfig) jobrun.Definition {
	d := jobrun.Definition{Provider: c.Provider, Repository: c.Repository, Ref: c.Ref, Timeout: c.Timeout, SecretTargets: append([]string(nil), c.SecretTargets...), Inputs: map[string]jobrun.Input{}}
	for name, input := range c.Inputs {
		d.Inputs[name] = jobrun.Input{Default: input.Default, Required: input.Required, Values: append([]string(nil), input.Values...)}
	}
	for _, run := range c.Runs {
		r := jobrun.RunDefinition{Name: run.Name, Workflow: run.Workflow, Inputs: run.Inputs, ResultArtifact: run.ResultArtifact}
		for _, image := range run.Images {
			r.Images = append(r.Images, jobrun.Image{Reference: image.Reference, Platforms: append([]string(nil), image.Platforms...)})
		}
		d.Runs = append(d.Runs, r)
	}
	return d
}

// A failed remote operation still has useful partial evidence. Always emit one
// complete document, then return a non-zero exit through the normal root path.
func printRemoteResult(result any, runErr error) error {
	envelope := struct {
		Result any    `json:"result"`
		Error  string `json:"error,omitempty"`
	}{Result: result}
	if runErr != nil {
		envelope.Error = runErr.Error()
	}
	return errors.Join(runErr, output.PrintJSON(envelope))
}
