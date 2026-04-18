package cli

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/kopia/kopia/repo"
	"github.com/kopia/kopia/snapshot"
	"github.com/kopia/kopia/snapshot/policy"
	"github.com/kopia/kopia/snapshot/upload"
)

type commandSnapshotCreateScheduled struct {
	sources  []string
	parallel int
	dryRun   bool
	verbose  bool
	quiet    bool

	svc advancedAppServices
	out textOutput
}

type sourceStatus struct {
	si       snapshot.SourceInfo
	nextTime time.Time
	lastTime time.Time
}

func (c *commandSnapshotCreateScheduled) setup(svc advancedAppServices, parent commandParent) {
	cmd := parent.Command("create-scheduled", "Run all snapshots that are due or overdue based on their schedule policies.").Alias("run-scheduled")
	cmd.Flag("source", "Source path to run (can be repeated)").StringsVar(&c.sources)
	cmd.Flag("parallel", "Number of snapshots to run in parallel").Default("1").IntVar(&c.parallel)
	cmd.Flag("dry-run", "Show what would run without executing").BoolVar(&c.dryRun)
	cmd.Flag("verbose", "Print information about all sources and their next scheduled times").BoolVar(&c.verbose)
	cmd.Flag("quiet", "Suppress output unless snapshots are actually run").BoolVar(&c.quiet)
	cmd.Action(svc.repositoryWriterAction(c.run))

	c.svc = svc
	c.out.setup(svc)
}

func (c *commandSnapshotCreateScheduled) run(ctx context.Context, rep repo.RepositoryWriter) error {
	sources, err := c.getSourcesToRun(ctx, rep)
	if err != nil {
		return err
	}

	if len(sources) == 0 {
		c.out.printStderr("No sources to run.\n")
		return nil
	}

	now := time.Now()
	var toRun []snapshot.SourceInfo

	var statuses []sourceStatus

	for _, si := range sources {
		lastSnapTime, err := c.getLastSnapshotTime(ctx, rep, si)
		if err != nil {
			return err
		}

		pol, _, _, err := policy.GetEffectivePolicy(ctx, rep, si)
		if err != nil {
			return err
		}

		nextTime, ok := pol.SchedulingPolicy.NextSnapshotTime(lastSnapTime, now)
		if !ok {
			continue
		}

		statuses = append(statuses, sourceStatus{si: si, nextTime: nextTime, lastTime: lastSnapTime})

		if nextTime.Before(now) || nextTime.Equal(now) {
			toRun = append(toRun, si)
		}
	}

	if c.verbose {
		c.printSourceStatus(statuses, now)
	}

	if len(toRun) == 0 {
		if !c.quiet {
			c.out.printStderr("No snapshots are due.\n")
		}
		return nil
	}

	if c.dryRun {
		c.out.printStderr("Would run %d snapshot(s):\n", len(toRun))
		for _, si := range toRun {
			c.out.printStderr("  %s\n", si)
		}
		return nil
	}

	if !c.quiet {
		c.out.printStderr("Running %d snapshot(s)...\n", len(toRun))
	}

	err = c.runSnapshots(ctx, rep, toRun)
	if err != nil {
		return err
	}

	if !c.quiet {
		c.out.printStderr("Completed %d snapshot(s).\n", len(toRun))
	}

	return nil
}

func (c *commandSnapshotCreateScheduled) printSourceStatus(statuses []sourceStatus, now time.Time) {
	if len(statuses) == 0 {
		return
	}

	c.out.printStderr("Source snapshots:\n")
	for _, st := range statuses {
		due := st.nextTime.Before(now) || st.nextTime.Equal(now)
		status := "OK"
		if due {
			status = "DUE"
		} else {
			delta := st.nextTime.Sub(now)
			status = "in " + delta.Round(time.Second).String()
		}
		c.out.printStderr("  %-60s %s (last: %s, next: %s)\n", st.si, status, formatTime(st.lastTime), formatTime(st.nextTime))
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Format("2006-01-02 15:04:05")
}

func (c *commandSnapshotCreateScheduled) getSourcesToRun(ctx context.Context, rep repo.Repository) ([]snapshot.SourceInfo, error) {
	if len(c.sources) > 0 {
		var result []snapshot.SourceInfo

		for _, s := range c.sources {
			si, err := snapshot.ParseSourceInfo(s, rep.ClientOptions().Hostname, rep.ClientOptions().Username)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to parse %q", s)
			}

			result = append(result, si)
		}

		return result, nil
	}

	allSources, err := snapshot.ListSources(ctx, rep)
	if err != nil {
		return nil, err
	}

	clientOpts := rep.ClientOptions()
	var localSources []snapshot.SourceInfo

	for _, si := range allSources {
		if si.Host == clientOpts.Hostname && si.UserName == clientOpts.Username {
			localSources = append(localSources, si)
		}
	}

	return localSources, nil
}

func (c *commandSnapshotCreateScheduled) getLastSnapshotTime(ctx context.Context, rep repo.Repository, si snapshot.SourceInfo) (time.Time, error) {
	manifests, err := snapshot.ListSnapshotManifests(ctx, rep, &si, nil)
	if err != nil {
		return time.Time{}, err
	}

	if len(manifests) == 0 {
		return time.Time{}, nil
	}

	snapshots, err := snapshot.LoadSnapshots(ctx, rep, manifests)
	if err != nil {
		return time.Time{}, err
	}

	if len(snapshots) == 0 {
		return time.Time{}, nil
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].StartTime.Before(snapshots[j].StartTime)
	})

	return snapshots[len(snapshots)-1].StartTime.ToTime(), nil
}

func (c *commandSnapshotCreateScheduled) runSnapshots(ctx context.Context, rep repo.RepositoryWriter, sources []snapshot.SourceInfo) error {
	semaphore := make(chan struct{}, c.parallel)

	var wg sync.WaitGroup

	c.svc.getProgress().StartShared()

	for _, si := range sources {
		wg.Add(1)

		semaphore <- struct{}{}

		go func(si snapshot.SourceInfo) {
			defer func() {
				<-semaphore
				wg.Done()
			}()

			if err := c.runSingleSnapshot(ctx, rep, si); err != nil {
				log(ctx).Errorf("unable to run snapshot for %v: %v", si, err)
			}
		}(si)
	}

	wg.Wait()
	c.svc.getProgress().FinishShared()

	return nil
}

func (c *commandSnapshotCreateScheduled) runSingleSnapshot(ctx context.Context, rep repo.RepositoryWriter, si snapshot.SourceInfo) error {
	log(ctx).Infof("Running snapshot for %v", si)

	previous, err := snapshot.FindPreviousManifests(ctx, rep, si, nil)
	if err != nil {
		return errors.Wrap(err, "unable to find previous manifests")
	}

	policyTree, err := policy.TreeForSource(ctx, rep, si)
	if err != nil {
		return errors.Wrap(err, "error getting policy tree")
	}

	uploader := upload.NewUploader(rep)
	uploader.Progress = c.svc.getProgress()

	entry, err := getLocalFSEntry(ctx, si.Path)
	if err != nil {
		return errors.Wrap(err, "unable to get local entry")
	}

	_, err = uploader.Upload(ctx, entry, policyTree, si, previous...)
	return err
}
