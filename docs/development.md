# Development Information

Generated API documentation is available on [pkg.go.dev](https:///pkg.go.dev/github.com/AliceO2Group/Control?tab=subdirectories/).

The release log is managed via [GitHub](https://github.com/AliceO2Group/Control/releases/).

Bugs go to [JIRA](https://its.cern.ch/jira/projects/OCTRL/issues).

## Release Procedure

1. Update documentation if necessary.
2. Bump `VERSION` file, commit, push (or pull request).
3. Run `hacking/release_notes.sh HEAD` to get a formatted commit message list since the last tag, copy it.
4. Paste the above into a [new GitHub release draft](https://github.com/AliceO2Group/Control/releases/new). Sort, categorize, add summary on top.
5. Pick a version number. Numbers `x.x.80`-`x.x.89` are reserved for Alpha pre-releases. Numbers `x.x.90`-`x.x.99` are reserved for Beta and RC pre-releases. If doing a pre-release, don't forget to tick `This is a pre-release`. When ready, hit `Publish release`.
6. Go to your local clone of `alidist`, ensure that the branch is `master` and that it's up to date. Then branch out into `aliecs-bump` (`git branch aliecs-bump`).
7. Bump the version in `control.sh`, `control-core.sh`, `control-occplugin.sh` and `coconut.sh`. Commit and push to `origin/aliecs-bump` (`git push -u origin aliecs-bump`).
8. Submit pull request with the above to `alisw/alidist`.
