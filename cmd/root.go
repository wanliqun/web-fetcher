package cmd

import (
	"fmt"
	"net/url"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/wanliqun/web-fetcher/fetcher"
	"github.com/wanliqun/web-fetcher/types"
)

var (
	// Used for flags.
	printMetadata bool

	rootCmd = &cobra.Command{
		Use:   "./fetch [--metadata | -a] <URL> [URL2] ...",
		Short: "CLI tool for web page scraping.",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
				return err
			}

			return validateArgs(args)
		},
		Run: run,
	}
)

func init() {
	rootCmd.Flags().BoolVarP(
		&printMetadata, "metadata", "a", false,
		"Print detailed metadata about fetched web pages",
	)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	fetcher := fetcher.NewFetcher(fetcher.Async())
	fetcher.OnFetched(func(result *types.FetchResult) {
		logger := logrus.WithField("URL", result.URL)

		if result.Err != nil {
			logger.WithError(result.Err).Error("Failed to fetch web page")
			return
		}

		if printMetadata && result.Metadata != nil {
			logger.WithFields(logrus.Fields{
				"numLinks":      result.Metadata.NumLinks,
				"images":        result.Metadata.NumImages,
				"lastFetchedAt": result.Metadata.LastFetchedAt,
			}).Info("Web page fetched")
		}
	})

	urlset := make(map[string]struct{})
	for i := range args {
		if _, ok := urlset[args[i]]; ok { // dedupe
			continue
		}

		urlset[args[i]] = struct{}{}
		fetcher.Fetch(args[i])
	}

	// Wait for all done.
	fetcher.Wait()
}

func validateArgs(args []string) error {
	for i := range args {
		_, err := url.Parse(args[i])
		if err != nil {
			return errors.Errorf("%s is not valid URL", args[i])
		}
	}

	return nil
}
