# Happybox

Happybox helps you organize your photos, videos and audios, all automatically.

- Having thousands of pictures and videos lost in complex nested folder structures?
- Don't you remember what camera or phone did you use to take that picture, or if it's even yours?
- Are you not sure if you have duplicates of the same picture?
- Do you have many screen shots mixed up with your regular photos?
- Do you have problems finding files from an specific date?

No problemo! Happybox will organize all the mess for you.

## Features

- Restructures a media folder recursively (pictures, videos, audios)
- Extracts media file metadata (like EXIF, XMP) and saves it in a metadata folder
- Organizes the media by year, camera and month
- Detects duplicates and stores them separately in a 'duplicates' folder
- Renames all the files using the timestamp and the file MD5 hash


## Requirements

- [go](https://github.com/golang/go)
- [exiftool](https://github.com/exiftool/exiftool) (tested on v11.16)


## Installation

```bash
go install github.com/itsjavi/happybox

```

## Usage

```bash
happybox [-limit n] [-ext "xxx|yyy|zzz"] [-dry-run] [-fix-dates] move|copy <src> [<dest>]

# example:

happybox -limit 100 -ext "jpg|png|gif" -fix-dates -dry-run copy ~/Pictures ./happybox-test

```

# To-Do

- Show total data copied (in MB or GB)
- Add info command (outputs metadata in json of one file or array)
-