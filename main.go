package main

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v2"
)

func main() {
	var app = &cli.App{
		Usage:                  "Media file organizer",
		Description:            "Organizes the image and video files of a folder recursively from source to destination.",
		ArgsUsage:              "source destination",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "dry-run",
				Value:   false,
				Aliases: []string{"d"},
				Usage:   "Do not process anything, just scan the directory and metadata.",
			},
			&cli.UintFlag{
				Name:    "limit",
				Value:   0,
				Aliases: []string{},
				Usage:   "Limit of files to process.",
			},
			&cli.StringFlag{
				Name:    "extensions",
				Value:   "",
				Aliases: []string{"ext"},
				Usage:   "Pipe-separated list of file extensions to process, e.g. \"jpg|mp4|mov\".",
			},
			&cli.BoolFlag{
				Name:    "convert-videos",
				Value:   false,
				Aliases: []string{"c"},
				Usage:   "Convert old video formats like 3gp, flv, mpeg, wmv, divx, etc. to MP4.",
			},
			&cli.BoolFlag{
				Name:    "fix-dates",
				Value:   false,
				Aliases: []string{"f"},
				Usage:   "Fix the file creation date by using the one in the metadata, if available.",
			},
			&cli.BoolFlag{
				Name:    "move",
				Value:   false,
				Aliases: []string{"m"},
				Usage:   "Move the files instead of copying them to the destination.",
			},
			&cli.BoolFlag{
				Name:    "quiet",
				Value:   false,
				Aliases: []string{"q"},
				Usage:   "It won't print anything, unless it's an error.",
			},
		},
		Action: func(c *cli.Context) error {
			params := CmdOptions{}

			if c.NArg() == 0 {
				return errors.New("Source and destination directory arguments are missing.")
			}
			if c.NArg() < 2 {
				return errors.New("Destination directory argument is missing.")
			}

			params.CurrentTime = time.Now()
			params.SrcDir, _ = filepath.Abs(c.Args().Get(0))
			params.DestDir, _ = filepath.Abs(c.Args().Get(1))
			params.DryRun = c.Bool("dry-run")
			params.Limit = c.Uint("limit")
			params.Extensions = c.String("extensions")
			params.ConvertVideos = c.Bool("convert-videos")
			params.FixDates = c.Bool("fix-dates")
			params.Move = c.Bool("move")
			params.Quiet = c.Bool("quiet")

			if !IsDir(params.SrcDir) {
				return errors.New("Source directory does not exist.")
			}

			if params.SrcDir == params.DestDir {
				return errors.New("Source and destination directories cannot be the same.")
			}

			_, err := TidyUp(params)

			return err
		},
	}
	err := app.Run(os.Args)
	HandleError(err)
}
