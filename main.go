package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/j-keck/plog"
	"github.com/j-keck/zfs-snap-diff/pkg/config"
	diffPkg "github.com/j-keck/zfs-snap-diff/pkg/diff"
	"github.com/j-keck/zfs-snap-diff/pkg/fs"
	"github.com/j-keck/zfs-snap-diff/pkg/scanner"
	"github.com/j-keck/zfs-snap-diff/pkg/zfs"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var version string = "SNAPSHOT"

type CliConfig struct {
	logLevel                  plog.LogLevel
	printVersion              bool
	scriptingOutput           bool
	snapshotTimemachineOutput bool
	coloredDiff               bool
	diffContextSize           int
}

func main() {
	zsdBin := os.Args[0]
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "zsd - cli tool to find older versions of a given file in your zfs snapshots.\n\n")
		fmt.Fprintf(os.Stderr, "USAGE:\n %s [OPTIONS] <FILE> <ACTION>\n\n", zsdBin)
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nACTIONS:\n")
		fmt.Fprintf(os.Stderr, "  list                           : list zfs snapshots where the given file was modified\n")
		fmt.Fprintf(os.Stderr, "  cat     <#|SNAPSHOT>           : show the file content from the given snapshot\n")
		fmt.Fprintf(os.Stderr, "  diff    <#|SNAPSHOT>           : show a diff from the selected snapshot to the current version\n")
		fmt.Fprintf(os.Stderr, "  revert  <#|SNAPSHOT> <CHUNK_NR>: revert the given chunk\n")
		fmt.Fprintf(os.Stderr, "  restore <#|SNAPSHOT>           : restore the file from the given snapshot\n")
		fmt.Fprintf(os.Stderr, "  grep    <PATTERN>              : grep changes\n")
		fmt.Fprintf(os.Stderr, "\nYou can use the snapshot number from the `list` output or the snapshot name to select a snapshot.\n")
		fmt.Fprintf(os.Stderr, "\nProject home page: https://j-keck.github.io/zsd\n")
	}

	initLogger()
	cliCfg := parseFlags()
	log := reconfigureLogger(cliCfg)

	if cliCfg.printVersion {
		fmt.Printf("zsd: %s\n", version)
		return
	}

	if len(flag.Args()) < 2 {
		fmt.Fprintf(os.Stderr, "Argument <FILE> <ACTION> missing (see `%s -h` for help)\n", zsdBin)
		return
	}

	// file path
	fileName := flag.Arg(0)
	filePath, err := filepath.Abs(fileName)
	if err != nil {
		log.Errorf("unable to get absolute path for: '%s' - %v", fileName, err)
		return
	}
	log.Debugf("full path: %s", filePath)

	// init zfs handler
	zfs, ds, err := zfs.NewZFSForFilePath(filePath)
	if err != nil {
		log.Errorf("unable to get zfs handler for path: '%s' - %v", filePath, err)
		return
	}
	log.Debugf("work on dataset: %s", ds.Name)

	// action
	action := flag.Arg(1)
	switch action {
	case "list":
		if !(cliCfg.scriptingOutput || cliCfg.snapshotTimemachineOutput) {
			fmt.Printf("scan the last %d days for other file versions\n", config.Get.DaysToScan)
		}

		dr := scanner.NDaysBack(config.Get.DaysToScan, time.Now())
		sc := scanner.NewScanner(dr, "auto", ds, zfs)
		scanResult, err := sc.FindFileVersions(filePath)
		if err != nil {
			log.Errorf("scan failed - %v", err)
			return
		}

		cacheFileVersions(scanResult.FileVersions)

		if cliCfg.snapshotTimemachineOutput {
			for idx, v := range scanResult.FileVersions {
				fmt.Printf("%d\t%s\t%s\t%s\n",
					idx, v.Snapshot.Name, v.Backup.Path, v.Backup.MTime.Format("Mon, 02 Jan 2006 15:04:05 -0700"))
			}
		} else if !cliCfg.scriptingOutput {

			// find the longest snapshot name to format the output table
			width := 0
			for _, v := range scanResult.FileVersions {
				width = int(math.Max(float64(width), float64(len(v.Snapshot.Name))))
			}

			// show snapshots where the file was modified
			header := fmt.Sprintf("%3s | %-12s | %-[3]*s | %12s", "#", "File changed", width, "Snapshot", "Snapshot age")
			fmt.Printf("%s\n%s\n", header, strings.Repeat("-", len(header)))
			for idx, v := range scanResult.FileVersions {
				mtimeAge := humanDuration(time.Since(v.Backup.MTime))
				snapAge := humanDuration(time.Since(v.Snapshot.Created))
				fmt.Printf("%3d | %12s | %[3]*s | %12s\n", idx, mtimeAge, width, v.Snapshot.Name, snapAge)
			}
		} else {
			for idx, v := range scanResult.FileVersions {
				fmt.Printf("%d\t%s\t%s\n", idx, v.Snapshot.Name, v.Snapshot.Created)
			}
		}

	case "cat":
		if len(flag.Args()) != 3 {
			fmt.Fprintf(os.Stderr, "Argument <#|SNAPSHOT> missing (see `%s -h` for help)\n", zsdBin)
			return
		}

		versionName := flag.Arg(2)
		version, err := lookupRequestedVersion(filePath, versionName)
		if err != nil {
			log.Error(err)
			return
		}

		file, err := fs.GetFileHandle(version.Backup.Path)
		if err != nil {
			log.Errorf("unable to find file in the snapshot - %v", err)
			return
		}

		content, err := file.ReadString()
		if err != nil {
			log.Errorf("unable to get content from %s - %v", file.Name, err)
			return
		}

		fmt.Println(content)

	case "diff":
		if len(flag.Args()) != 3 {
			fmt.Fprintf(os.Stderr, "Argument <#|SNAPSHOT> missing (see `%s -h` for help)\n", zsdBin)
			return
		}

		versionName := flag.Arg(2)
		version, err := lookupRequestedVersion(filePath, versionName)
		if err != nil {
			log.Error(err)
			return
		}

		diff, err := diffPkg.NewDiffFromPath(version.Backup.Path, filePath, cliCfg.diffContextSize)
		if err != nil {
			log.Errorf("unable to create diff - %v", err)
			return
		}

		if !cliCfg.scriptingOutput {
			fmt.Printf("Diff from the actual version to the version from: %s\n", version.Backup.MTime)
		}

		fmt.Printf("%s", diffsPrettyText(diff, cliCfg.coloredDiff))

	case "revert":
		if len(flag.Args()) != 4 {
			fmt.Fprintf(os.Stderr, "Argument <#|SNAPSHOT> and / or <CHUNK_NR> missing (see `%s -h` for help)\n", zsdBin)
			return
		}

		versionName := flag.Arg(2)
		version, err := lookupRequestedVersion(filePath, versionName)
		if err != nil {
			log.Error(err)
			return
		}

		diff, err := diffPkg.NewDiffFromPath(version.Backup.Path, filePath, cliCfg.diffContextSize)
		if err != nil {
			log.Errorf("unable to create diff - %v", err)
			return
		}

		chunkNr, err := strconv.Atoi(flag.Arg(3))
		if err != nil {
			log.Errorf("given chunk-nr was not a number - %v", err)
			return
		}

		if chunkNr < 0 || chunkNr >= len(diff.Deltas) {
			log.Errorf("given chunk-nr was out of range - valid range: 0..%d", len(diff.Deltas)-1)
			return
		}
		deltas := diff.Deltas[chunkNr]

		backupPath, err := version.Current.Backup()
		if err != nil {
			log.Errorf("unable to backup the current version - %v", err)
			return
		}
		if !cliCfg.scriptingOutput {
			fmt.Printf("backup from the actual version created at: %s\n", backupPath)
		}

		// revert the given chunk
		err = diffPkg.PatchPath(filePath, deltas)
		if err != nil {
			log.Errorf("unable to revert chunk-nr: %d - err: %v", chunkNr, err)
			return
		}

		if !cliCfg.scriptingOutput {
			fmt.Printf("reverted: \n%s", diffPrettyText(deltas, cliCfg.coloredDiff))
		}

	case "restore":
		if len(flag.Args()) != 3 {
			fmt.Fprintf(os.Stderr, "Argument <#|SNAPSHOT> missing (see `%s -h` for help)\n", zsdBin)
			return
		}

		versionName := flag.Arg(2)
		version, err := lookupRequestedVersion(filePath, versionName)
		if err != nil {
			log.Error(err)
			return
		}

		backupPath, err := version.Current.Backup()
		if err != nil {
			log.Errorf("unable to backup the current version - %v", err)
			return
		}
		if !cliCfg.scriptingOutput {
			fmt.Printf("backup from the actual version created at: %s\n", backupPath)
		}

		// restore the backup version
		version.Backup.Copy(version.Current.Path)

		if !cliCfg.scriptingOutput {
			fmt.Printf("version restored from snapshot: %s\n", version.Snapshot.Name)
		}

	case "grep":
		if len(flag.Args()) != 3 {
			fmt.Fprintf(os.Stderr, "Argument <PATTERN> missing (see `%s -h` for help)\n", zsdBin)
			return
		}

		pattern := strings.ToLower(flag.Arg(2))

		if !cliCfg.scriptingOutput {
			fmt.Printf("scan the last %d days for other file versions\n", config.Get.DaysToScan)
		}

		dr := scanner.NDaysBack(config.Get.DaysToScan, time.Now())
		sc := scanner.NewScanner(dr, "auto", ds, zfs)
		scanResult, err := sc.FindFileVersions(filePath)
		if err != nil {
			log.Errorf("scan failed - %v", err)
			return
		}

		cacheFileVersions(scanResult.FileVersions)

		maxSnapNameWidth := 0
		if !cliCfg.scriptingOutput {
			// find the longest snapshot name to format the output table
			for _, v := range scanResult.FileVersions {
				maxSnapNameWidth = int(math.Max(float64(maxSnapNameWidth), float64(len(v.Snapshot.Name))))
			}

			header := fmt.Sprintf("%3s | %-12s | %-[3]*s | %12s | %5s | %-33s", "#",
				"File changed", maxSnapNameWidth, "Snapshot", "Snapshot age", "Line", "Change")
			fmt.Printf("%s\n%s\n", header, strings.Repeat("-", len(header)))
		}

		for versionIdx, version := range scanResult.FileVersions {
			a := version.Backup.Path
			b := filePath
			if versionIdx > 0 {
				b = scanResult.FileVersions[versionIdx - 1].Backup.Path
			}
			bFh, err := fs.GetFileHandle(b)
			if err != nil {
				log.Error(err)
				return
			}


			diff, err := diffPkg.NewDiffFromPath(a, b, cliCfg.diffContextSize)
			if err != nil {
				log.Errorf("unable to create diff - %v", err)
				return
			}

			for _, chunks := range diff.Deltas {
				for _, delta := range chunks {
					if delta.Type == diffPkg.Del || delta.Type == diffPkg.Ins {
						lines := strings.Split(delta.Text, "\n")
						for lineOffset, line := range lines {
							if strings.Contains(strings.ToLower(line), pattern) {
								var change string
								switch delta.Type {
								case diffPkg.Del: change = "-"
								case diffPkg.Ins: change = "+"
								}

								mTimeAge := humanDuration(time.Since(bFh.MTime))
								snapAge := humanDuration(time.Since(version.Snapshot.Created))
								ln := delta.LineNrFrom + lineOffset
								line := strings.TrimSpace(line)

								if !cliCfg.scriptingOutput {
									fmt.Printf("%3d | %12s | %[3]*s | %12s | %5d | %1s %s\n",
										versionIdx, mTimeAge, maxSnapNameWidth, version.Snapshot.Name, snapAge, ln, change, line)
								} else {
									mTime := bFh.MTime.Format("Mon, 02 Jan 2006 15:04:05 -0700")
									snapTime := version.Snapshot.Created.Format("Mon, 02 Jan 2006 15:04:05 -0700")
									fmt.Printf("%3d\t%12s\t%s\t%s\t%d\t%1s %s\n",
										versionIdx, mTime, version.Snapshot.Name, snapTime, ln, change, line)
								}
							}
						}
					}
				}
			}

		}

	default:
		fmt.Fprintf(os.Stderr, "invalid action: %s (see `%s -h` for help)\n", action, zsdBin)
		return
	}
}

func lookupRequestedVersion(filePath, versionName string) (*scanner.FileVersion, error) {

	// load file-versions from cache file
	fileVersions, err := loadCachedFileVersions()
	if err != nil {
		return nil, err
	}

	// `versionName` can be the snapshot number from the `list` output or the name
	var version *scanner.FileVersion
	if idx, err := strconv.Atoi(versionName); err == nil {
		if idx >= 0 && idx < len(fileVersions) {
			version = &fileVersions[idx]
		} else {
			return nil, errors.New("snapshot number not found")
		}
	} else {
		for _, v := range fileVersions {
			if v.Snapshot.Name == versionName {
				version = &v
				break
			}
		}
		if version == nil {
			return nil, errors.New("snapshot name not found")
		}
	}

	// verify the cache is for the requested file
	if version.Current.Path != filePath {
		return nil, errors.New("invalid cache - initialize cache with `zsd <FILE> list")
	}

	// verify the file exists (maybe the snapshot was deleted after the `list` action)
	switch _, err := os.Stat(version.Backup.Path); err.(type) {
	case nil:
		return version, nil
	case *os.PathError:
		return nil, errors.New("obsolete cache - reload cache with `zsd <FILE> list`")
	default:
		return nil, err
	}
}

func humanDuration(dur time.Duration) string {
	s := int(dur.Seconds())
	if s < 60 {
		return fmt.Sprintf("%d seconds", s)
	}

	m := int(dur.Minutes())
	if m < 60 {
		return fmt.Sprintf("%d minutes", m)
	}
	h := int(dur.Hours())
	if h < 48 {
		return fmt.Sprintf("%d hours", h)
	}

	d := int(h / 24)
	return fmt.Sprintf("%d days", d)
}

func parseFlags() CliConfig {
	loadConfig()

	cliCfg := new(CliConfig)

	// cli
	flag.BoolVar(&cliCfg.printVersion, "V", false, "print version and exit")
	flag.IntVar(&config.Get.DaysToScan, "d", config.Get.DaysToScan, "days to scan")
	flag.BoolVar(&cliCfg.scriptingOutput, "H", false,
		"Scripting mode. Do not print headers, print absolute dates and separate fields by a single tab")
	flag.BoolVar(&cliCfg.snapshotTimemachineOutput, "snapshot-timemachine", false,
		"Special output for Snapshot-timemachine (https://github.com/mrBliss/snapshot-timemachine)")

	var noColoredDiff bool
	flag.BoolVar(&noColoredDiff, "no-color", false,
		"Don't use colored diff output use '+' / '-' for inserts / removed lines")

	flag.IntVar(&cliCfg.diffContextSize, "diff-context-size", config.Get.DiffContextSize,
		"show N lines before and after each diff")

	// logging
	cliCfg.logLevel = plog.Note
	plog.FlagDebugVar(&cliCfg.logLevel, "v", "debug output")
	plog.FlagTraceVar(&cliCfg.logLevel, "vv", "trace output with caller location")

	// zfs
	zfsCfg := &config.Get.ZFS
	flag.BoolVar(&zfsCfg.UseSudo, "use-sudo", zfsCfg.UseSudo, "use sudo when executing 'zfs' commands")
	flag.BoolVar(&zfsCfg.MountSnapshots, "mount-snapshots", zfsCfg.MountSnapshots,
		"mount snapshot (only necessary if it's not mounted by zfs automatically)")

	flag.Parse()
	cliCfg.coloredDiff = !noColoredDiff
	return *cliCfg
}

func loadConfig() {
	plog.DropUnhandledMessages()
	configDir, _ := fs.ConfigDir()
	configPath := configDir.Path + "/zfs-snap-diff.toml"
	config.LoadConfig(configPath)
}

func initLogger() {
	consoleLogger := plog.NewConsoleLogger(" | ")
	consoleLogger.AddLogFormatter(plog.Level)
	consoleLogger.AddLogFormatter(plog.Message)

	plog.GlobalLogger().Add(consoleLogger)
}

func reconfigureLogger(cliCfg CliConfig) plog.Logger {

	consoleLogger := plog.NewConsoleLogger(" | ")
	consoleLogger.SetLevel(cliCfg.logLevel)
	consoleLogger.AddLogFormatter(plog.Level)

	if cliCfg.logLevel == plog.Trace {
		consoleLogger.AddLogFormatter(plog.Location)
	}

	consoleLogger.AddLogFormatter(plog.Message)

	return plog.GlobalLogger().Reset().Add(consoleLogger)
}
