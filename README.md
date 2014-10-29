DashMe
======

DashMe is a frontend for DASH content generated, if necessary, on the fly from any supported format.

Supported Formats
-----------------

* Classic MP4
* SmoothStreaming
* DASH

Installing
----------

Requirements:
* Go installed on the machine
* Go project workspace set-up

```
> cd $GOPATH/src/github.com/canalplus
> git clone git@github.com:canalplus/DashMe.git
> cd $GOPATH
> go install github.com/canalplus/DashMe
```

Usage
-----

```
Usage of ./bin/DashMe:
  -cache="/tmp/DashMe": Directory used for caching
  -port="3000": TCP port used when starting the API
  -video="/home/aubin/Workspace/videos/": Directory containing the videos
```
