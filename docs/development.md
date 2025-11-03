# Development Information

Generated API documentation is available on [pkg.go.dev](https:///pkg.go.dev/github.com/AliceO2Group/Control?tab=subdirectories/).

The release log is managed via [GitHub](https://github.com/AliceO2Group/Control/releases/).

Bugs go to [JIRA](https://its.cern.ch/jira/projects/OCTRL/issues).

## Release Procedure

### Major releases

1. Update documentation if necessary.
2. Bump `VERSION` file, commit, push (or pull request).
3. Go to a [new GitHub release draft](https://github.com/AliceO2Group/Control/releases/new). Use "Generate release notes" to create a list of changes. Write a short summary at the top.
4. Go to your local clone of `alidist`, ensure that the branch is `master` and that it's up to date. Then branch out into `aliecs-bump` (`git branch aliecs-bump`).
5. Bump the version in `control.sh`, `control-core.sh`, `control-occplugin.sh` and `coconut.sh`. Commit and push to `origin/aliecs-bump` (`git push -u origin aliecs-bump`).
6. Submit pull request with the above to `alisw/alidist`.

### Patch releases

1. Update documentation if necessary.
2. If the patch release should NOT be based on master:

  * Checkout the tag which the patch release should be based on, e.g. `git checkout v1.34.0`
  * Create a branch called `branch_<planned_tag>`, e.g. `git checkout -b branch_v1.34.1`.
  * Cherry-pick desired commits

3. Bump `VERSION` file, commit, push (or pull request).
4. Go to a [new GitHub release draft](https://github.com/AliceO2Group/Control/releases/new). Use "Generate release notes" to create a list of changes. Write a short summary at the top.
5. Go to your local clone of `alidist`, ensure that the branch is `master` and that it's up to date. Then branch out into `aliecs-bump` (`git branch aliecs-bump`).
6. Bump the version in `control.sh`, `control-core.sh`, `control-occplugin.sh` and `coconut.sh`. Commit and push to `origin/aliecs-bump` (`git push -u origin aliecs-bump`).
7. Submit pull request with the above to `alisw/alidist`.

