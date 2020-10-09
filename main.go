package main

import (
	"errors"
	"fmt"
	tm "github.com/buger/goterm"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v2"
)

func RunCommand(c *cli.Context) {
	if c.NArg() == 0 {
		return errors.New("source and destination directory arguments are missing")
	}
	if c.NArg() < 2 {
		return errors.New("destination directory argument is missing")
	}

	ctx := CreateAppContext(c)
	fileMetaChan := make(chan FileMeta)

	go TidyRoutine(ctx, &ctx.stats, fileMetaChan)
	for {
		meta, isOk := <-fileMetaChan
		if isOk == false {
			break // channel closed
		}
		PrintAppStats(meta.Path, ctx.stats, ctx)
	}

	PrintAppStats("--", ctx.stats, ctx)
	fmt.Println()
	PrintLn(tm.Color("Took %s", tm.BLUE), time.Since(ctx.StartTime))
}

func CreateAppContext(c *cli.Context) AppContext {
	var err error
	ctx := AppContext{}

	ctx.StartTime = time.Now()

	ctx.SrcDir, err = filepath.Abs(c.Args().First())
	Catch(err)

	ctx.DestDir, err = filepath.Abs(c.Args().Get(1))
	Catch(err)

	ctx.DryRun = c.Bool("dryrun")
	ctx.Limit = c.Int("limit")
	ctx.CustomExtensions = c.String("extensions")
	ctx.CustomMediaType = c.String("type")
	ctx.CustomExclude = c.String("exclude")
	ctx.FixCreationDates = c.Bool("fixdates")
	ctx.CreateDbOnly = c.Bool("dbonly")
	ctx.CreateThumbnails = c.Bool("thumbnails")
	ctx.MoveFiles = c.Bool("move")
	ctx.Quiet = c.Bool("quiet")
	ctx.Stats = AppRunStats{}

	if !IsDir(ctx.SrcDir) {
		return errors.New("source directory does not exist")
	}

	if ctx.SrcDir == ctx.DestDir {
		return errors.New("source and destination directories cannot be the same")
	}
}

func CreateCliApp() *cli.App {
	var app = &cli.App{
		Usage:                  "Media file organizer",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "dryrun",
				Value:   false,
				Aliases: []string{"d"},
				Usage:   "Do not process anything, just scan the directory and metadata.",
			},
			&cli.BoolFlag{
				Name:    "quiet",
				Value:   false,
				Aliases: []string{"q"},
				Usage:   "It won't print anything, unless it's an error.",
			},
		},
		Commands: []*cli.Command{
			{
				Name:        "run",
				Usage:       "Organizes source folder media into a destination folder.",
				Description: "Organizes the image and video files of a folder recursively from source to destination.",
				ArgsUsage:   "source destination",
				Flags: []cli.Flag{
					&cli.UintFlag{
						Name:    "limit",
						Value:   0,
						Aliases: []string{"l"},
						Usage:   "Limit of files to process.",
					},
					&cli.StringFlag{
						Name:    "extensions",
						Value:   "",
						Aliases: []string{"ext"},
						Usage:   "Custom pipe-separated list of file extensions to process, e.g. \"jpg|mp4|mov|docx\".",
					},
					&cli.StringFlag{
						Name:  "type",
						Value: "",
						Usage: "Custom media type to tag the custom extensions with, e.g. \"document\".",
					},
					&cli.BoolFlag{
						Name:    "fixdates",
						Value:   false,
						Aliases: []string{"f"},
						Usage:   "Fix the file creation date by using the one in the metadata, if available.",
					},
					&cli.BoolFlag{
						Name:  "dbonly",
						Value: false,
						Usage: "Only created the DB index, without moving or copying files.",
					},
					&cli.BoolFlag{
						Name:  "thumbnails",
						Value: false,
						Usage: "Create thumbnails for the compatible images and videos.",
					},
					&cli.BoolFlag{
						Name:    "move",
						Value:   false,
						Aliases: []string{"m"},
						Usage:   "Move the files instead of copying them to the destination.",
					},
					&cli.StringFlag{
						Name:  "exclude",
						Value: "",
						Usage: "Custom pipe-separated list of path patterns to exclude, e.g. \"Screenshot\"",
					},
				},
				Action: func(c *cli.Context) error {
					RunCommand(c)
					return nil
				},
			},
			{
				Name:      "reindex",
				Usage:     "Reconciles the metadata DB entries with the actual files, fixing the missing ones.",
				ArgsUsage: "dir",
				Action: func(c *cli.Context) error {
					ctx := CreateAppContext(c)

					if !ctx.HasMetadataDb() {
						return errors.New("databases/metadata.sqlite DB does not exist under the given path")
					}

					return nil
				},
			},
			{
				Name: "create-thumbnails",
				Action: func(c *cli.Context) error {
					return nil
				},
			},
			{
				Name: "detect-faces",
				Action: func(c *cli.Context) error {
					return nil
				},
			},
			{
				Name: "reindex",
				Action: func(c *cli.Context) error {
					return nil
				},
			},
			//{
			//	Name: "fixdb",
			//	Action: func(c *cli.Context) error {
			//		targetDir := c.Args().First()
			//
			//		if targetDir == "" || !IsDir(targetDir) {
			//			return errors.New("The given directory does not exist or it is not a directory.")
			//		}
			//
			//		ctx := AppContext{SrcDir: targetDir, DestDir: targetDir}
			//		ctx.InitDb()
			//		result := ctx.Db.db.Exec("UPDATE files SET path = REPLACE(path, '/.','.')")
			//		Catch(result.Error)
			//		return nil
			//	},
			//},
		},
	}
	return app
}

func main() {
	_, timeErr := time.LoadLocation("UTC")
	Catch(timeErr)

	app := CreateCliApp()
	err := app.Run(os.Args)

	Catch(err)
}
