# Development Information

Generated API documentation is available on [pkg.go.dev](https:///pkg.go.dev/github.com/AliceO2Group/Control?tab=subdirectories/).

The release log is managed via [GitHub](https://github.com/AliceO2Group/Control/releases/).

Bugs go to [JIRA](https://alice.its.cern.ch/jira/browse/OCTRL).

## Release Procedure

1) Update documentation if necessary.
2) Bump `VERSION` file, commit.
3) Run `hacking/release_notes.sh HEAD` to get a formatted commit message list since the last tag, copy it.
4) Paste the above into a [new GitHub release draft](https://github.com/AliceO2Group/Control/releases/new). Sort, categorize, add summary on top.
5) Pick a version number. Numbers `x.x.80`-`x.x.89` are reserved for Alpha pre-releases. Numbers `x.x.90`-`x.x.99` are reserved for Beta and RC pre-releases. If doing a pre-release, don't forget to tick `This is a pre-release`. When ready, hit `Publish release`.
6) Go to your local clone of `alice-o2-flp-suite-documentation`, descend into `docs/aliecs`. `git pull --rebase` to ensure the submodule points to the tag created just now. Commit and push (or merge request).
7) Go to your local clone of `alidist`, ensure that the branch is `master` and that it's up to date. Then branch out into `aliecs-bump` (`git branch aliecs-bump`).
8) Bump the version in `control.sh`, `control-core.sh`, `control-occplugin.sh` and `coconut.sh`. Commit and push to `origin/aliecs-bump` (`git push -u origin aliecs-bump`).
9) Submit pull request with the above to `alisw/alidist`.
10) Go to [Jenkins](https://alijenkins.cern.ch/job/BuildRPM). Accept the certificate warning and find a previous build of `Control` or `O2Suite`. Pick it, and rebuild either `Control` or both, always with `DEFAULTS` set to `o2-dataflow` and `ALIDIST_SLUG` set to `alisw/alidist@aliecs-bump`.
11) Go to your local copy of `system-configuration`, open file `ansible/roles/basevars/vars/main.yml`. Bump what's relevant, commit, push, merge request.