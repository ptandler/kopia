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

type commandSnapshotRunScheduled struct {
	sources         []string
	parallel       int
	dryRun         bool

	svc advancedAppServices
	out textOutput
}

func (c *commandSnapshotRunScheduled) setup(svc advancedAppServices, parent commandParent) {
	cmd := parent.Command("run-scheduled", "Run all snapshots that are due or overdue based on their schedule policies.")
	cmd.Flag("source", "Source path to run (can be repeated)").StringsVar(&c.sources)
	cmd.Flag("parallel", "Number of snapshots to run in parallel").Default("1").IntVar(&c.parallel)
	cmd.Flag("dry-run", "Show what would run without executing").BoolVar(&c.dryRun)
	cmd.Action(svc.repositoryWriterAction(c.run))

	c.svc = svc
	c.out.setup(svc)
}

func (c *commandSnapshotRunScheduled) run(ctx context.Context, rep repo.RepositoryWriter) error {
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

		if nextTime.Before(now) || nextTime.Equal(now) {
			toRun = append(toRun, si)
		}
	}

	if len(toRun) == 0 {
		c.out.printStderr("No snapshots are due.\n")
		return nil
	}

	if c.dryRun {
		c.out.printStderr("Would run %d snapshot(s):\n", len(toRun))
		for _, si := range toRun {
			c.out.printStderr("  %s\n", si)
		}
		return nil
	}

	return c.runSnapshots(ctx, rep, toRun)
}

func (c *commandSnapshotRunScheduled) getSourcesToRun(ctx context.Context, rep repo.Repository) ([]snapshot.SourceInfo, error) {
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

	return snapshot.ListSources(ctx, rep)
}

func (c *commandSnapshotRunScheduled) getLastSnapshotTime(ctx context.Context, rep repo.Repository, si snapshot.SourceInfo) (time.Time, error) {
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

func (c *commandSnapshotRunScheduled) runSnapshots(ctx context.Context, rep repo.RepositoryWriter, sources []snapshot.SourceInfo) error {
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

func (c *commandSnapshotRunScheduled) runSingleSnapshot(ctx context.Context, rep repo.RepositoryWriter, si snapshot.SourceInfo) error {
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