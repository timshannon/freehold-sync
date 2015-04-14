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
A *Sync Profile* consists of Local directory location, and a Remote directory location on a freehold instance.  The Profile also describes:

Direction:
	* Both - Syncs files both to the remote and the local locations
	* Remote Only - Only syncs files to the remote location
	* Local Only - Only syncs files to the local location

Conflict Resolution - If a file is modified both at the local and remote locations with *X* amount of seconds, then
	* Overwrite the older file with the newer one or
	* Rename the older file with a timestamp and copy in the new one

Ignore List - List of regular expressions that when matched to a files full path, will skip the syncing on that file.  By default an ignore list entry is added to ignore hidden files (i.e files that start ".").

Local changes are captured via filesystem events.  Freehold sync will poll the changing file waiting for it's size and modified date to stop changing, then queue up the file for syncing.

Remote changes are polled for on a regular basis (default every 30 seconds, configurable via the settings.json file).  That *snapshot* of a remote folder is stored in a local datastore, and compared against on the next remote poll.  The differences are accumulated, and queued up for syncing.  This is how freehold-sync determines if a remote file has been deleted, or just doesn't exist, and queues up the proper change for syncing.

Syncing consists of comparing the modified date on freehold instance to the modified date on the local file.  For this reason, it is important for you to be running the latest version of Freehold which provides a method for preserving a file's original modified date upon upload.

Sync changes can come at any time, and enter out of order (e.g. someone just deleted the parent folder of the file currently queued for syncing), so occasionally order of operation errors will occur.  Those errors will be queued up and retried 3 times.  After 3 failures, they will get logged in the error log.

The freehold-sync web interface will keep track of the last time you viewed the errors tab, and you'll see an indicator on the tab when new, yet unseen errors exist.

settings.json
-----------------------
settings.json is a json formated file that can be used to change how freehold-sync runs. When freehold-sync first starts, it will print out a list of possible settings.json locations in order of priority (first location gets higher priority over settings files in any lower location).  It will also print out where the currently used settings.json file is located.

The most likely default locations for this file will be:
	* Linux -  `"/home/<username>/.config/freehold-sync/settings.json"`
	* Windows - `"\users\<username>\AppData\Roaming\"`

It is in this settings.json file in which you can set the port freehold-sync runs on (by default 6080) and the remote polling frequency (30 seconds).
