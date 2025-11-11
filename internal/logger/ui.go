package logger

import (
	"fmt"
	"log/slog"
	"os"
)

// PrintScannerBanner prints a scanner-specific banner
func PrintScannerBanner() {
	banner := "═══════════════════════════════════════════════════════════════════════════════════════════════════════\n" +
		"                              🔍  WELCOME TO Go-SSV-Scanner  🔍            \n\n" +
		"                ____            ____ ______     __   ____                                  \n" +
		"               / ___| ___      / ___/ ___\\ \\   / /  / ___|  ___ __ _ _ __  _ __   ___ _ __ \n" +
		"              | |  _ / _ \\ ____\\___ \\___ \\\\ \\ / /___\\___ \\ / __/ _` | '_ \\| '_ \\ / _ \\ '__|\n" +
		"              | |_| | (_) |_____|__) |__) |\\ V /_____|__) | (_| (_| | | | | | | |  __/ |   \n" +
		"               \\____|\\___/     |____/____/  \\_/     |____/ \\___\\__,_|_| |_|_| |_|\\___|_|   \n" +
		"                                                                                            \n" +
		"   	  +----------------------------------------------------------------------------------+\n" +
		"   	  |  Tool for retrieving events data (cluster snapshots, owner nonce & operator      |\n" +
		"   	  |  public keys) from the SSV network contract.                                     |\n" +
		"   	  +----------------------------------------------------------------------------------+\n\n" +
		"═══════════════════════════════════════════════════════════════════════════════════════════════════════\n"
	fmt.Fprint(os.Stderr, banner)
}

// PrintBanner prints a simple banner to stderr
func PrintBanner(title string) {
	banner := fmt.Sprintf("********************************\n  %s\n********************************\n", title)
	fmt.Fprint(os.Stderr, banner)
}

// Structured log helpers using slog (stderr and file via GlobalInit)
func LogNonce(l *slog.Logger, owner string, nonce int) {
	l.Info("next nonce", "owner", owner, "nonce", nonce)
}

func LogCluster(l *slog.Logger, owner, block, operators, data string) {
	l.Info("cluster snapshot", "owner", owner, "block", block, "operators", operators, "data", data)
}

func LogOperatorSaved(l *slog.Logger, path string, count int) {
	l.Info("operator data saved", "path", path, "count", count)
}

// PrintKeyValueTable prints a simple key-value table to stderr with a title.
func PrintKeyValueTable(title string, rows [][2]string) {
	if title != "" {
		fmt.Fprintf(os.Stderr, "\n%s\n", title)
	}
	// find max key width
	maxk := 0
	for _, r := range rows {
		if len(r[0]) > maxk {
			maxk = len(r[0])
		}
	}
	for _, r := range rows {
		fmt.Fprintf(os.Stderr, "%-*s : %s\n", maxk, r[0], r[1])
	}
}
