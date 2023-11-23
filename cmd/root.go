package cmd

import (
	"fmt"
	"os"

	"github.com/PuerkitoBio/purell"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/wanliqun/web-fetcher/fetcher"
	"github.com/wanliqun/web-fetcher/types"
)

var (
	// Used for flags.
	printMetadata bool
	mirror        bool
	verbose       bool

	rootCmd = &cobra.Command{
		Use:   "./fetch [--metadata | -a] [--mirror | -m] [--verbose | -v] <URL> [URL2] ...",
		Short: "CLI tool for web page scraping.",
		Args:  cobra.MinimumNArgs(1),
		Run:   run,
	}
)

func init() {
	rootCmd.Flags().BoolVarP(
		&printMetadata, "metadata", "a", false,
		"Print detailed metadata about fetched web pages",
	)

	rootCmd.Flags().BoolVarP(
		&mirror, "mirror", "m", false,
		"Download web page assets for local mirror",
	)

	rootCmd.Flags().BoolVarP(
		&verbose, "verbose", "v", false, "Verbose output",
	)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	options := []fetcher.FetcherOption{fetcher.Async()}
	if mirror {
		options = append(options, fetcher.Mirror())
	}
	fetcher := fetcher.NewFetcher(options...)

	fetcher.OnFetched(func(result *types.FetchResult) {
		logger := logrus.WithField("URL", result.URL)

		if result.Err != nil {
			logger.WithError(result.Err).Error("Failed to fetch web page")
			return
		}

		if printMetadata && result.Metadata != nil {
			logger = logger.WithFields(logrus.Fields{
				"numLinks":      result.Metadata.NumLinks,
				"images":        result.Metadata.NumImages,
				"lastFetchedAt": result.Metadata.LastFetchedAt,
			})
		}

		logger.Info("Web page fetched")
	})

	urlSet := make(map[string]struct{})
	for i := range args {
		// Normalized passed in URL argument at first.
		normURL, err := purell.NormalizeURLString(args[i], purell.FlagsSafe)
		if err != nil {
			logrus.WithField("URL", args[i]).
				WithError(err).
				Fatalln("Failed to normalize URL")
		}

		if _, ok := urlSet[normURL]; !ok { // dedupe
			urlSet[normURL] = struct{}{}
		}
	}

	// Start fetching
	for u := range urlSet {
		fetcher.Fetch(u)
	}

	// Wait for all done.
	fetcher.Wait()
}
