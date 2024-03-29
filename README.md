# _mediatidy_

Command-line tool written in Go to organise all media files in a directory recursively by date, detecting duplicates.

## Features

- Organizes media (images and videos) by year and month folders.
- Extracts metadata like EXIF and XMP into separated JSON files.
- Detects duplicates (by comparing file checksum) and skips moving/copying them.
- Normalizes the file names.
- Fixes file creation time, by using the one in the metadata if available.

## Requirements

- [go >= v1.19](https://github.com/golang/go)
- [exiftool >= v12](https://github.com/exiftool/exiftool)
- ffmpeg


## Installation

```bash

go install github.com/itsjavi/mediatidy/cmd/mediatidy@latest

```

## Usage

Check all the available options with the help command:

```bash

mediatidy --help

```
