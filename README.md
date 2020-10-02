# _mediatidy_

Command-line tool written in Go to organise all media files in a directory recursively by date, detecting duplicates.

## Features

- Organizes media (images and videos) by year, month and day folders.
- Extracts metadata like EXIF and XMP in separated JSON files.
- Detects duplicates and avoids using them, having priority the older files to be less destructive.
- Normalizes the file names.
- Fixes file creation time, by using the one in the metadata if available.

### Planned

- Convert old video formats to MP4 (H.264 + AAC) while tidying up using [ffmpeg >= 4.2](https://ffmpeg.org/).
- Add support for graceful command termination with signals and routines.

## Requirements

- [go >= v1.15](https://github.com/golang/go)
- [exiftool >= v11.80](https://github.com/exiftool/exiftool)


## Installation

```bash

go install github.com/itsjavi/mediatidy

```

## Usage

Check all the available options with the help command:

```bash

mediatidy --help

```
