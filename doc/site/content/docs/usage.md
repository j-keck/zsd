+++
title = "Usage"
draft = false
creator = "Emacs 26.3 (Org mode 9.1.9 + ox-hugo)"
weight = 30
+++

## List snapshots {#list-snapshots}

Use the `list` action to list all snapshots where the
given file was modified.

`zsd` lists only snapshots where the given file was modified.
You can adjust the number of days to scan with the `-d` flag.

```text
 main⟩ zsd main.go list
scan the last 2 days for other file versions
  # | File changed | Snapshot                                 | Snapshot age
----------------------------------------------------------------------------
  0 |   20 minutes | zfs-auto-snap_frequent-2020-07-12-14h00U |   18 minutes
  1 |   34 minutes | zfs-auto-snap_frequent-2020-07-12-13h45U |   33 minutes
  2 |   49 minutes | zfs-auto-snap_frequent-2020-07-12-13h30U |   48 minutes
```


## Show file content {#show-file-content}

Use the `cat` action to show the file content from
the given snapshot.

{{< hint info >}}
You can use the snapshot number from the `list` output
or the snapshot name to select a snapshot.
{{< /hint >}}

```text
 # i use `head` here to keep the example short
 main⟩ zsd main.go cat 2 | head -5
package main

import (
	"errors"
	"flag"
```


## Show diff {#show-diff}

To show a diff from the selected snapshot to the actual version
use the `diff` action.

{{< hint info >}}
You can use the snapshot number from the `list` output
or the snapshot name to select a snapshot.
{{< /hint >}}

```text
 # i use `head` here to keep the example short
 main⟩ zsd -no-color -diff-context-size 2 main.go diff 1  | head -30
Diff from the actual version to the version from: 2020-07-12 15:44:43.566374368 +0200 CEST
=============================
Chunk 0 - starting at line 43
-----------------------------
        fmt.Fprintf(os.Stderr, "  revert  <#|SNAPSHOT> <CHUNK_NR>: revert the given chunk\n")
		fmt.Fprintf(os.Stderr, "  restore <#|SNAPSHOT>           : restore the file from the given snapshot\n")
-       fmt.Fprintf(os.Stderr, "  grep    <PATTERN>              : grep changed lines\n")
+       fmt.Fprintf(os.Stderr, "  grep    <PATTERN>              : grep changes\n")
        fmt.Fprintf(os.Stderr, "\nYou can use the snapshot number from the `list` output or the snapshot name to select a snapshot.\n")
		fmt.Fprintf(os.Stderr, "\nProject home page: https://j-keck.github.io/zsd\n")

==============================
Chunk 1 - starting at line 264
------------------------------
        pattern := strings.ToLower(flag.Arg(2))

+       if !cliCfg.scriptingOutput {
+           fmt.Printf("scan the last %d days for other file versions\n", config.Get.DaysToScan)
+       }

        dr := scanner.NDaysBack(config.Get.DaysToScan, time.Now())
		sc := scanner.NewScanner(dr, "auto", ds, zfs)

==============================
Chunk 2 - starting at line 272
------------------------------
        }
```


## Revert a change {#revert-a-change}

You can revert a single change with the `revert` action.


#### View a diff to the backup version {#view-a-diff-to-the-backup-version}

```text
main⟩ zsd -diff-context-size 1 -no-color cache.go diff 0
Diff from the actual version to the version from: 2020-07-04 14:29:37.807643286 +0200 CEST
=============================
Chunk 0 - starting at line 13
-----------------------------
	j, err := json.Marshal(versions)
+	println(err)
	if err != nil {

=============================
Chunk 1 - starting at line 18
-----------------------------
	cacheDir, err := fs.CacheDir()
+	println(err)
	if err != nil {
```


#### Revert a single change {#revert-a-single-change}

Use the chunk-nr from the `diff` output to select the change to revert.
See the ****Chunk &lt;NR&gt;**** line to get it.

```text
main⟩ ./zsd -diff-context-size 1 -no-color cache.go revert 0 1
backup from the actual version created at: /home/j/.cache/zfs-snap-diff/backups/home/j/prj/priv/zfs-snap-diff/zsd/cache.go_20200704_144023
reverted:
	cacheDir, err := fs.CacheDir()
+	println(err)
	if err != nil {
```


#### Check the result {#check-the-result}

```text
main⟩ zsd -diff-context-size 1 -no-color cache.go diff 0
Diff from the actual version to the version from: 2020-07-04 14:29:37.807643286 +0200 CEST
=============================
Chunk 0 - starting at line 13
-----------------------------
	j, err := json.Marshal(versions)
+	println(err)
	if err != nil {
```

{{< hint warning >}}
A backup of the current version will be created.
{{< /hint >}}


## Restore file {#restore-file}

To restore a given file with an older version use `restore`.

{{< hint info >}}
You can use the snapshot number from the `list` output
or the snapshot name to select a snapshot.
{{< /hint >}}

```text
main⟩ zsd go.mod restore 0
backup from the actual version created at: /home/j/.cache/zfs-snap-diff/backups/home/j/prj/priv/zfs-snap-diff/go.mod_20200212_182709%
version restored from snapshot: zfs-auto-snap_hourly-2020-02-12-12h00U
```

{{< hint warning >}}
A backup of the current version will be created.
{{< /hint >}}


## Grep changes {#grep-changes}

You can search changes with `grep <PATTERN>`. This search only changed lines.

To view the whole diff use the snapshot number or snapshot name from the output.

```text
 main⟩ zsd main.go grep trim
scan the last 2 days for other file versions
  # | File changed | Snapshot                                 | Snapshot age |  Line | Change
------------------------------------------------------------------------------------------------------------------------
  3 |      1 hours |   zfs-auto-snap_hourly-2020-07-12-13h00U |      2 hours |   285 | + line := strings.TrimSpace(line)
```


## Flags {#flags}

Use the `-h` flag to see the supported flags.

```text
main⟩ zsd -h
zsd - cli tool to find older versions of a given file in your zfs snapshots.

USAGE:
 ./zsd [OPTIONS] <FILE> <ACTION>

OPTIONS:
  -H	Scripting mode. Do not print headers, print absolute dates and separate fields by a single tab
  -V	print version and exit
  -d int
        days to scan (default 2)
 -mount-snapshots
        mount snapshot (only necessary if it's not mounted by zfs automatically)
 -snapshot-timemachine
        Special output for Snapshot-timemachine (https://github.com/mrBliss/snapshot-timemachine)
 -use-sudo
        use sudo when executing 'zfs' commands
  -v	debug output
  -vv
        trace output with caller location

ACTIONS:
  list                           : list zfs snapshots where the given file was modified
  cat     <#|SNAPSHOT>           : show the file content from the given snapshot
  diff    <#|SNAPSHOT>           : show a diff from the selected snapshot to the current version
  revert  <#|SNAPSHOT> <CHUNK_NR>: revert the given chunk
  restore <#|SNAPSHOT>           : restore the file from the given snapshot
  grep    <PATTERN>              : grep changes

You can use the snapshot number from the `list` output or the snapshot name to select a snapshot.

Project home page: https://j-keck.github.io/zsd
```
