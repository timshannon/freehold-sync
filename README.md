Freehold Sync
===================
Freehold Sync is a tool for synchronizing files between the local computer and one or more [freehold](https://bitbucket.org/tshannon/freehold) instances.

When first run a light, local webserver will run on port 6080.  You can access it at by going to [http://localhost:6080](http://localhost:6080) in your browser, or opening it from the system tray.  The port can be changed in the settings.json file. The location of which will be outputted when freehold-sync is first run. 

From there you can setup Sync Profiles which synchronize a local folder with a freehold instance.  You can have more than one sync profile, and you can sync against multiple freehold instances.  Sync Profiles can also be set to only sync one direction, or ignore certain files.  


Getting Started
--------------------
You can download one of the pre-compiled binaries from the downloads page.  Currently binaries only exist for Windows and Linux.  Freehold-Sync should build fine on a mac, but I do not have access to one currently to build on so binaries aren't available. Place the executable somewhere on your local computer, and if you want it to automatically start, you can put a link in your *Start Up* folder on windows or set it as a startup application according to your distriubution of linux.

Once running, you'll need to create a new *Sync Profile*.  A sync profile describes which folders (and their sub-directories) to keep in sync across your local machine and a freehold instance.  The sync profile also describes how those files should be synchronized (see synchronization details below for more information).

Building from Source
---------------------
In order to build Freehold-Sync from source you'll need a standard [Go installation](http://golang.org/doc/install), as well as the capability to do a [CGO build](http://blog.golang.org/c-go-cgo).  This is necessary to build the platform specific system tray handling.

### Windows
To do cgo builds, you will need to install MinGW. In order to prevent the terminal window from appearing when your application runs, build with:

```go build -ldflags -H=windowsgui```

### Linux
In addition to the essential GNU build tools, you will need to have the GTK+ 3.0 development headers, and the App Indicator development headers installed.

On Ubuntu and derivitives, that would look like this:
```
sudo apt-get install build-essential

sudo apt-get install libgtk-3-dev

sudo apt-get install libappindicator3-dev

```

### Mac OSX
You'll need the "Command Line Tools for Xcode", which can be installed using Xcode. You should be able to run the cc command from a terminal window.


Synchronization Details
-----------------------------
