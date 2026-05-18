// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/paths"
)

var followLogs bool

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View the OpenTelemetry Collector log output",
	Long: `Prints the collector log file contents. By default, shows the last 50 lines.
Use --follow to continuously stream new log lines.`,
	Args: cobra.NoArgs,
	RunE: runLogs,
}

func init() {
	logsCmd.Flags().BoolVarP(&followLogs, "follow", "f", false, "continuously stream new log lines")
	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, _ []string) error {
	logPath := paths.LogFilePath()

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		fmt.Fprintln(cmd.OutOrStdout(), "No log file found.")
		return nil
	}

	if followLogs {
		return tailFollow(cmd, logPath)
	}

	return printLastLines(cmd, logPath, 50)
}

// printLastLines reads the file and prints the last n lines to stdout.
func printLastLines(cmd *cobra.Command, path string, n int) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer f.Close()

	// Read all lines into a ring buffer of size n.
	scanner := bufio.NewScanner(f)
	lines := make([]string, 0, n)
	for scanner.Scan() {
		if len(lines) >= n {
			lines = append(lines[1:], scanner.Text())
		} else {
			lines = append(lines, scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading log file: %w", err)
	}

	out := cmd.OutOrStdout()
	for _, line := range lines {
		fmt.Fprintln(out, line)
	}
	return nil
}

// tailFollow opens the log file, prints the last 50 lines, then continuously
// polls for new content until interrupted by SIGINT/SIGTERM.
func tailFollow(cmd *cobra.Command, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer f.Close()

	// Print last 50 lines first.
	if err := printLastLines(cmd, path, 50); err != nil {
		return err
	}

	// Seek to end of file for tailing.
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("seeking to end of log file: %w", err)
	}

	// Set up signal handling for clean exit.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	out := cmd.OutOrStdout()
	reader := bufio.NewReader(f)

	for {
		select {
		case <-sigCh:
			return nil
		default:
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				fmt.Fprint(out, line)
			}
			if err != nil {
				// No new data available, sleep and retry.
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
}
