package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
	"github.com/spf13/cobra"
)

type resourceChildExit struct{ code int }

func (e *resourceChildExit) Error() string { return fmt.Sprintf("child exited %d", e.code) }

type resourceReport struct {
	Schema            string  `json:"schema"`
	PolicyFingerprint string  `json:"policy_fingerprint"`
	Class             string  `json:"class"`
	Priority          string  `json:"priority"`
	WaitDuration      *string `json:"wait_duration"`
	ExecutionDuration *string `json:"execution_duration"`
	Outcome           string  `json:"outcome"`
	ExitCode          *int    `json:"exit_code"`
	ClaimID           string  `json:"claim_id,omitempty"`
}

var resourceStateDirectory = resourceadmission.DefaultDirectory

func newResourceCommand() *cobra.Command {
	var policyPath, class, priority, waitLimit, executionLimit, reportPath string
	resource := &cobra.Command{Use: "resource", Short: "Coordinate cooperative host resources", Long: "Coordinate classified local workloads using the host resource policy.", SilenceUsage: true, SilenceErrors: true}
	run := &cobra.Command{Use: "run -- COMMAND [ARG...]", Short: "Run a classified command", Long: "Wait for a cooperative host resource lease and run the command. An interrupted active lease remains quarantined for operator review.", Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if os.Getenv("DVA_RESOURCE_SUPERVISED") != "" || os.Getenv("DVA_CI_PARENT_RUN") != "" {
			return errors.New("nested resource admission is unsupported")
		}
		if policyPath == "" {
			policyPath = os.Getenv("DVA_RESOURCE_POLICY")
		}
		p, digest, err := resourceadmission.ReadPolicy(policyPath)
		if err != nil {
			return err
		}
		if class != "heavy" && class != "light" {
			return errors.New("resource class must be heavy or light")
		}
		if priority != "interactive" && priority != "normal" && priority != "background" {
			return errors.New("invalid resource priority")
		}
		wait, err := time.ParseDuration(waitLimit)
		if err != nil || wait <= 0 {
			return errors.New("wait timeout must be positive")
		}
		execution, err := time.ParseDuration(executionLimit)
		if err != nil || execution <= 0 {
			return errors.New("execution timeout must be positive")
		}
		dir, err := resourceStateDirectory()
		if err != nil {
			return err
		}
		var report *os.File
		if reportPath != "" {
			report, err = resourceadmission.CreateReport(reportPath)
			if err != nil {
				return err
			}
			defer report.Close()
		}
		result := resourceReport{Schema: "dva-resource-run/1", PolicyFingerprint: digest, Class: class, Priority: priority, Outcome: "waiting"}
		writeReport := func() error {
			if report == nil {
				return nil
			}
			if err := json.NewEncoder(report).Encode(result); err != nil {
				return err
			}
			return report.Sync()
		}
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		waitCtx, cancel := context.WithTimeout(ctx, wait)
		started := time.Now()
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "resource waiting class=%s priority=%s\n", class, priority)
		waitDone := make(chan struct{})
		go func() {
			defer close(waitDone)
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-waitCtx.Done():
					return
				case <-ticker.C:
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "resource waiting class=%s elapsed=%s\n", class, time.Since(started).Round(time.Second))
				}
			}
		}()
		lease, err := resourceadmission.Acquire(waitCtx, dir, p, digest, class, priority)
		cancel()
		<-waitDone
		waitDuration := time.Since(started).String()
		result.WaitDuration = &waitDuration
		if err != nil {
			result.Outcome = "admission_failed"
			return errors.Join(err, writeReport())
		}
		result.ClaimID = lease.ID()
		execCtx, execCancel := context.WithTimeout(ctx, execution)
		defer execCancel()
		child := exec.Command(args[0], args[1:]...)
		child.WaitDelay = time.Second
		for _, value := range os.Environ() {
			if !strings.HasPrefix(value, "DVA_RESOURCE_SUPERVISED=") {
				child.Env = append(child.Env, value)
			}
		}
		child.Env = resourceadmission.CapWorkerEnvironment(append(child.Env, "DVA_RESOURCE_SUPERVISED=1"), p.WorkerLimits[class])
		resourceConfigureProcess(child)
		child.Stdin = cmd.InOrStdin()
		child.Stdout = cmd.OutOrStdout()
		child.Stderr = cmd.ErrOrStderr()
		started = time.Now()
		childWaited := false
		err = child.Start()
		if err == nil {
			waited := make(chan error, 1)
			go func() { waited <- child.Wait() }()
			select {
			case err = <-waited:
				childWaited = true
			case <-execCtx.Done():
				select {
				case err = <-waited:
					childWaited = true
				default:
					termErr := resourceTermGroup(child.Process.Pid)
					select {
					case err = <-waited:
						childWaited = true
					case <-time.After(750 * time.Millisecond):
						killErr := resourceKillGroup(child.Process.Pid)
						select {
						case err = <-waited:
							childWaited = true
						case <-time.After(2 * time.Second):
							err = errors.New("child wait did not complete; lease retained")
						}
						err = errors.Join(err, killErr)
					}
					if childWaited && !resourceGroupGone(child.Process.Pid) {
						time.Sleep(750 * time.Millisecond)
					}
					if !resourceGroupGone(child.Process.Pid) {
						killErr := resourceKillGroup(child.Process.Pid)
						if killErr != nil {
							fmt.Fprintf(cmd.ErrOrStderr(), "resource group kill failed: %v\n", killErr)
						}
						err = errors.Join(err, killErr)
					}
					if termErr != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "resource group terminate failed: %v\n", termErr)
					}
					err = errors.Join(err, termErr)
				}
			}
		}
		if child.Process != nil {
			executionDuration := time.Since(started).String()
			result.ExecutionDuration = &executionDuration
		}
		code := 0
		var exitErr *exec.ExitError
		if err != nil && errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			code = 124
			result.Outcome = "timed_out"
		} else if errors.As(err, &exitErr) {
			code = resourceSignalExit(exitErr)
			result.Outcome = "child_failed"
		} else if err != nil {
			result.Outcome = "start_failed"
		} else {
			result.Outcome = "succeeded"
		}
		if child.Process != nil && (err == nil || exitErr != nil || code == 124) {
			result.ExitCode = &code
		}
		if child.Process == nil || (childWaited && resourceGroupGone(child.Process.Pid)) {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Second)
			releaseErr := lease.Complete(cleanupCtx)
			cleanupCancel()
			if releaseErr != nil {
				err = errors.Join(err, releaseErr)
				fmt.Fprintf(cmd.ErrOrStderr(), "resource lease release failed: %v\n", releaseErr)
			}
		} else {
			result.Outcome = "quarantined"
			if quarantineErr := lease.Quarantine(); quarantineErr != nil {
				err = errors.Join(err, quarantineErr)
			}
			quarantineErr := fmt.Errorf("resource lease %s quarantined: child process group may remain; operator review required", lease.ID())
			err = errors.Join(err, quarantineErr)
			fmt.Fprintln(cmd.ErrOrStderr(), quarantineErr)
		}
		if reportErr := writeReport(); reportErr != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "resource report failed: %v\n", reportErr)
			if code == 0 {
				return reportErr
			}
		}
		if code != 0 {
			return &resourceChildExit{code: code}
		}
		return err
	}}
	run.Flags().StringVar(&policyPath, "policy", "", "Absolute host policy path (or DVA_RESOURCE_POLICY)")
	run.Flags().StringVar(&class, "class", "", "Resource class: heavy or light")
	run.Flags().StringVar(&priority, "priority", "normal", "Priority: interactive, normal or background")
	run.Flags().StringVar(&waitLimit, "wait-timeout", "5m", "Maximum admission wait")
	run.Flags().StringVar(&executionLimit, "execution-timeout", "", "Maximum child execution time")
	run.Flags().StringVar(&reportPath, "report-file", "", "New private report path")
	var statusPolicy string
	status := &cobra.Command{Use: "status", Short: "Inspect cooperative resource claims", Long: "Read current waiting and running claims. Interrupted active claims remain quarantined for operator review.", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if statusPolicy == "" {
			statusPolicy = os.Getenv("DVA_RESOURCE_POLICY")
		}
		_, digest, err := resourceadmission.ReadPolicy(statusPolicy)
		if err != nil {
			return err
		}
		dir, err := resourceStateDirectory()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), time.Second)
		defer cancel()
		entries, err := resourceadmission.Status(ctx, dir, digest)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(entries)
	}}
	status.Flags().StringVar(&statusPolicy, "policy", "", "Absolute host policy path (or DVA_RESOURCE_POLICY)")
	resource.AddCommand(run, status)
	resource.GroupID = "core"
	return resource
}

func init() { rootCmd.AddCommand(newResourceCommand()) }
