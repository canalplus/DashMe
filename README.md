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
* gccgo and make installed on the machine

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
