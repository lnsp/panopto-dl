# panopto-dl

Video downloader for TUM-hosted Panopto.

## Usage

> You need Go to build this software and youtube-dl to run it.

```
$ git clone https://github.com/lnsp/panopto-dl
$ cd panopto-dl
$ go build
$ ./panopto-dl -a "YOUR_ASPX_TOKEN" -id "STREAM_ID"
```

You can get the ASPX token from the cookie viewer (cookie `.ASPXAUTH`) inside your browser after logging into Panopto.
The stream ID can be retrieved from the video URL.

```
Example URL: https://tum.cloud.panopto.eu/Panopto/Pages/Viewer.aspx?id=fa6eb567-bc50-4023-b736-ad4700911ea9

The stream ID is the id part of the URL.
```
