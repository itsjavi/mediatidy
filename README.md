# desktop-tidy

😊Command line tool that helps you classifying and organizing automagically your photos, videos, audios and documents ✨

- Having thousands of pictures and videos lost in complex nested folder structures?
- Don't you remember what camera or phone did you use to take that picture, or if it's even yours?
- Are you not sure if you have duplicates of the same picture?
- Do you have many screen shots mixed up with your regular photos?
- Do you have problems finding files from an specific date?

No problemo! This tool will organize all that mess for you.

## Features

- Restructures the content of a folder recursively (pictures, videos, audios, archives, contacts, documents, ...)
- Extracts media file metadata (like EXIF, XMP) and saves it in a metadata folder
- Organizes the media by year, camera / app and month
- Detects screenshots (by path name)
- Detects duplicates and stores them separately in a 'duplicates' folder
- Renames all the files using the timestamp and the file MD5 hash


## Requirements

- [go](https://github.com/golang/go)
- [exiftool](https://github.com/exiftool/exiftool) (tested on v11.16)


## Installation

```bash
go install github.com/itsjavi/desktop-tidy

```

## Usage

```bash
desktop-tidy [-limit n] [-ext "xxx|yyy|zzz"] [-dry-run] [-fix-dates] move|copy <src> [<dest>]

# example:

desktop-tidy -limit 100 -ext "jpg|png|gif" -fix-dates -dry-run copy ~/Pictures ./desktop-tidy-test

```
