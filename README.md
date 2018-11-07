# _happytimes_

😊Command line tool that helps you classifying and organizing automagically your photos, videos, audios and documents ✨

- Having thousands of pictures and videos lost in infinitely nested folder structures?
- Don't you remember what camera or phone did you use to take that picture, or if it's even yours?
- Are you not sure if you have duplicates of the same picture?
- Do you have many screen shots mixed up with your regular photos?
- Do you have problems finding files from an specific date?

No problem! _happytimes_ will organize all the mess for you. Rediscover all your data.

## Features

- Restructures a folder recursively (pictures, videos, audios, documents, contacts, archives, ...)
- Extracts media file metadata (like EXIF, XMP) and saves it in a metadata folder
- Organizes the media by year, camera / app and month
- Detects duplicates and stores them separately in a 'duplicates' folder
- Slugizes the name of all the files and adds the original creation timestamp and the file MD5 hash


## Requirements

- [go >= v1.10](https://github.com/golang/go)
- [exiftool (>= v11.10](https://github.com/exiftool/exiftool)


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