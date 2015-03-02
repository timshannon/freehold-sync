Freehold Sync
===================
Freehold Sync is a tool for synchronizing files between the local computer and one or more [freehold](https://bitbucket.org/tshannon/freehold) instances.

When first run a light, local webserver will run on port 6080.  You can access it at [http://localhost:6080](http://localhost:6080).  The port can be changed in the settings.json file. The location of which will be outputted when freehold-sync is first run. 

From there you can setup Sync Profiles which synchronize a local folder with a freehold instance.  You can have more than one sync profile, and you can sync against multiple freehold instances.  Sync Profiles can also be set to sync one way, or not to sync deletes.  You can also specific methods for file conflict resolution.
