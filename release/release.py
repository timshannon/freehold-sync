#!/usr/bin/env python

# I'm not smart enough to figure out the debian packaging tools so I wrote this
# based on this: http://ubuntuforums.org/showthread.php?t=910717
# build a deb file from the contents of the deb folder in the running dir

import sys
import subprocess
import os
import shutil

APP_NAME = "freehold-sync"
DESCRIPTION ="An application for synchronizing files with your desktop and a freehold instance"
AUTHOR = "Tim Shannon <shannon.timothy@gmail.com>"
PATH_TO_EXE = "../"

arch = "amd64"

def makeDebFolder(version, packageRevision):
    name = debFileName(version, packageRevision)
    shutil.copytree("deb", name)

    metaPath = os.path.join(name, "DEBIAN")
    if not os.path.exists(metaPath):
        os.mkdir(metaPath)

    with open(os.path.join(metaPath, "control"), "w") as metaFile:
       metaFile.writelines(["Package: "+APP_NAME, "\n",
            "Version: "+version+"-"+str(packageRevision), "\n",
            "Section: base", "\n",
            "Priority: optional", "\n",
            "Architecture: "+arch, "\n",
            "Installed-Size: "+str(os.path.getsize(name)/1024), "\n",
            "Depends:", "\n",
            "Maintainer: "+AUTHOR, "\n",
            "Description: " + DESCRIPTION, "\n"])


def makeDeb(version):
    packageRevision = 0

    #check if version is already packaged, if so, increase package revision

    name = debFileName(version, packageRevision)
    while os.path.exists(name+".deb"):
        packageRevision+=1
        name = debFileName(version, packageRevision)

    makeDebFolder(version, packageRevision)

    #copy in latest build
    debExePath = os.path.join(name, "usr/bin")
    os.makedirs(debExePath)
    shutil.copy(os.path.join(PATH_TO_EXE, APP_NAME), debExePath)

    #set proper ownership
    subprocess.call(["chown", "-R", "root:root", name])

    print "Making deb file " + name +".deb"
    subprocess.call(["dpkg-deb", "--build", name])
    
    shutil.rmtree(name)

def debFileName(version, packageRevision):
    return APP_NAME+"_"+version+"-"+str(packageRevision) +"_"+arch


def makeTarball(version):
    print "making tarball"    
    name = APP_NAME+"_"+version+"_linux_"+arch
    os.mkdir(name)

    #copy in latest build
    shutil.copy(os.path.join(PATH_TO_EXE, APP_NAME), name)
    
    subprocess.call(["tar", "czf", name+".tar.gz", name])

    shutil.rmtree(name)


##### main

#version passed in 
if len(sys.argv) > 1:
    version = sys.argv[1]
    arch = subprocess.check_output(["dpkg-architecture", "-qDEB_BUILD_ARCH"]).strip()
    makeDeb(version)        
    makeTarball(version)
 
    print "Releases built successfully"
else:
    print "Please specify a version: <major>.<minor>, e.g. 1.3"    
