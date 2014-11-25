DashMe
======

DashMe is a frontend for DASH content generated, if necessary, on the fly from any supported format.

Supported Formats
-----------------

Any format supported by FFMPEG is supported by DashMe as FFMPEG is used for parsing and demuxing.
The only requirement is that the codec used is H264.

We can however cite some format :
* MP4/MOV
* AVI
* MKV
* SmoothStreaming
* DASH
* ...

Installing
----------

Requirements:
* GCCGO installed
* make installed
* gcc installed
* go installed (for cgo)
* FFMPEG (or LibAV) only for libavutil and libavformat

```
> git clone git@github.com:canalplus/DashMe.git
> cd DashMe
> ./configure
> make
```

Usage
-----

```
Usage of ./bin/DashMe:
  -cache="/tmp/DashMe": Directory used for caching
  -port="3000": TCP port used when starting the API
  -video="/home/aubin/Workspace/videos/": Directory containing the videos
```

REST Interface
--------------

Route                 | Method | Behaviour
----------------------|--------|--------------------------------------------------
/files                | GET    | Return : {name, proto, path, isLive, generated}
/files                | POST   | Add an element for generation
/files/upload         | POST   | Upload a file and add it for generation
/dash/<name>/generate | POST   | Start generation of a file/stream
/dash/<name>/generate | DELETE | Stop generation of chunks/manifest for live only
/dash/<name>/<elm>    | GET    | Return file (chunk or manifest)
