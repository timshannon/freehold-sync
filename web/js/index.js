// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

$(document).ready(function() {
    Ractive.DEBUG = false;
    var r = new Ractive({
        el: "main",
        template: "#tMain",
        data: {
            alerts: [],
            page: "main",
            logPage: 0,
        }
    });

    loadProfiles();

    r.on({
        "addAlert": function(type, lead, detail) {
            if (!type) {
                type = "danger";
            }

            if (!lead) {
                lead = "An error occurred!";
            }
            r.push("alerts", {
                type: type,
                lead: lead,
                detail: detail
            });
            $("body").addClass("modal-open");
        },
        "dismissAlert": function(event) {
            r.splice(event.keypath.split(".")[0], event.index.i, 1);
            var alerts = r.get("alerts");
            if (alerts.length === 0) {
                $("body").removeClass("modal-open");
            }
        },
        "loadLogs": function() {
            loadLogs();
        },
        "logPageNext": function(event) {
            r.add("logPage", 1);
            loadLogs();
        },
        "logPagePrev": function(event) {
            r.subtract("logPage", 1);
            if (r.get("logPage") < 0) {
                r.set("logpage", 0);
            }
            loadLogs();
        },
        "loadProfiles": function() {
            loadProfiles();
        },
        "editProfile": function(event) {
            $.ajax({
                    type: "get",
                    url: "/profile/",
                    dataType: "json",
                    data: JSON.stringify(event.context),
                })
                .done(function(result) {
                    r.set("currentProfile", new Profile(result.data));
                    r.set("ignoreInsert", "");
                    r.set("page", "editProfile");
                })
                .fail(function(result) {
                    error(result);
                });

        },
        "newProfile": function() {
            r.set("currentProfile", new Profile());
            r.set("ignoreInsert", "");
            r.set("page", "newProfile");
        },
        "cancelProfile": function() {
            r.set("page", "main");
        },
        "saveNewProfile": function(event) {
            var profile = event.context;
            profile.saveNew()
                .done(function() {
                    r.set("page", "main");
                    loadProfiles();
                })
                .fail(function(result) {
                    r.set("currentProfile.profileError", result.responseJSON.message);
                });
        },
        "saveProfile": function(event) {
            event.context.save()
                .done(function() {
                    r.set("page", "main");
                    loadProfiles();
                })
                .fail(function(result) {
                    r.set("currentProfile.profileError", result.responseJSON.message);
                });
        },
        "deleteProfile": function(event) {
            event.context.delete()
                .done(function() {
                    loadProfiles();
                    r.set("page", "main");
                })
                .fail(function(result) {
                    error(result);
                });
        },
        "toggleDirection": function(event) {
            if (event.context.direction < 2) {
                event.context.direction++;
            } else {
                event.context.direction = 0;
            }
            r.set(event.keypath + ".direction", event.context.direction);
        },
        "showLocalModal": function(event) {
            $("#localModal").modal("show");
            r.set("selectKeypath", "");
            getPath("", "root.children");
        },
        "showRemoteModal": function(event) {
            r.set("clientError", null);
            r.set("selectKeypath", "");
            r.set("remoteClientSet", false);
            var c = new Client(r.get("currentProfile.client"));
            if (c.url) {
                c.connect();
            }

            $("#remoteModal").modal("show");
        },
        "treeopen": function(event) {
            var openPath = event.keypath + ".open";
            r.toggle(openPath);
            if (r.get(openPath)) {
                getPath(event.context.path, event.keypath + ".children", event.context.client);
            }
        },
        "treeselect": function(event) {
            r.set("selectKeypath", event.keypath);
        },
        "treeUpDir": function(event) {
            var root = r.get("root.path");
            root = root.split("/");
            if (root.length <= 1) {
                return;
            }

            root.pop();
            var newPath = root.join("/");
            if (newPath === "") {
                newPath = "/";
            }
            r.set("root.path", newPath);
            r.set("root.name", newPath);
            r.set("root.open", true);
            getPath(newPath, "root.children");
        },
        "selectLocal": function() {
            var keypath = r.get("selectKeypath");
            r.set("currentProfile.localPath", r.get(keypath + ".path"));
            $("#localModal").modal("hide");
        },
        "selectRemote": function() {
            var keypath = r.get("selectKeypath");
            r.set("currentProfile.remotePath", r.get(keypath + ".path"));
            $("#remoteModal").modal("hide");
        },
        "setRemoteClient": function(event) {
            event.original.preventDefault();
            var c = event.context.client;
			var err = r.get("clientError");
            if (err && err.indexOf("Invalid user and / or password") != -1 && c.token) {
                c.token = "";
            }
            c.connect();

        },
        "showHiddenClick": function(event) {
            getPath("", "root.children");
        },
        "addIgnore": function(event) {
            r.set("currentProfile.ignoreErr", null);
            var newIgnore = r.get("ignoreInsert");
            if (!newIgnore) {
                return;
            }
            try {
                var regex = new RegExp(newIgnore);
            } catch (e) {
                r.set("currentProfile.ignoreErr", "Invalid Regex!");
                return;
            }


            var ignores = r.get("currentProfile.ignore");
            ignores.push(newIgnore);
            r.set("ignoreInsert", "");
        },
        "removeIgnore": function(event) {
            var s = event.keypath.split(".");
            s.pop();

            r.splice(s.join("."), event.index.i, 1);
        },
    });




    function Profile(profile) {
        if (!profile) {
            this.id = "";
            this.name = "";
            this.direction = 0;
            this.conflictResolution = 0;
            this.conflictDurationSeconds = 0;
            this.active = true;
            this.ignore = [];
            this.localPath = "";
            this.remotePath = "";
            this.client = new Client();
        } else {
            /*this = profile;*/
            this.id = profile.id;
            this.name = profile.name;
            this.direction = profile.direction;
            this.conflictResolution = profile.conflictResolution;
            this.conflictDurationSeconds = profile.conflictDurationSeconds;
            this.active = profile.active;
            this.ignore = profile.ignore;
            this.localPath = profile.localPath;
            this.remotePath = profile.remotePath;
            this.client = new Client(profile.client);

        }
        //methods
        this.saveNew = function() {
            this.conflictDurationSeconds = Number(this.conflictDurationSeconds);
            return $.ajax({
                type: "POST",
                url: "/profile/",
                dataType: "json",
                data: JSON.stringify(this),
            });
        };
        this.save = function() {
            this.conflictDurationSeconds = Number(this.conflictDurationSeconds);
            return $.ajax({
                type: "PUT",
                url: "/profile/",
                dataType: "json",
                data: JSON.stringify(this),
            });
        };
        this.delete = function() {
            this.conflictDurationSeconds = Number(this.conflictDurationSeconds);
            return $.ajax({
                type: "DELETE",
                url: "/profile/",
                dataType: "json",
                data: JSON.stringify(this),
            });
        };
    }

    function Client(client) {
        if (!client) {
            this.url = "";
            this.user = "";
            this.password = "";
            this.token = "";
        } else {
            this.url = client.url;
            this.user = client.user;
            this.password = client.password;
            this.token = client.token;
        }
        this.getToken = function(name) {
            return $.ajax({
                type: "POST",
                url: "/remote/token/",
                dataType: "json",
                data: JSON.stringify({
                    name: name,
                    client: this
                }),
            });
        };
        this.connect = function() {

            if (!this.token && !r.get("skipToken")) {
                this.getToken(r.get("currentProfile.name"))
                    .done(function(result) {
                        this.token = result.data.token;
                        this.password = "";
                        this.connect();
                    }.bind(this))
                    .fail(function(result) {
                        r.set("clientError", result.responseJSON.message);
                    });
                return;
            }

            $.ajax({
                    type: "get",
                    url: "/remote/root/",
                    dataType: "json",
                    data: JSON.stringify({
                        client: this,
                    }),
                })
                .done(function(result) {
                    r.set("root", {
                        name: result.data,
                        path: result.data,
                        open: true,
                        client: this,
                    });

                    getPath(result.data, "root.children", this);
                    r.set("remoteClientSet", true);
                    r.set("clientError", null);
                }.bind(this))
                .fail(function(result) {
                    r.set("clientError", result.responseJSON.message);
                }.bind(this));
        };
    }


    function loadProfiles() {
        $.ajax({
                type: "get",
                url: "/profile/",
                dataType: "json",
            })
            .done(function(result) {
                r.set("profiles", result.data);
            })
            .fail(function(result) {
                error(result);
            });
    }

    function loadLogs(type) {
        $.ajax({
                type: "get",
                url: "/log/",
                dataType: "json",
                data: JSON.stringify({
                    page: r.get("logPage"),
                    type: type,
                }),
            })
            .done(function(result) {
                var logs = result.data;

                for (var i = 0; i < logs.length; i++) {
                    logs[i].when = new Date(logs[i].when).toLocaleString();
                }

                r.set("logs", logs);
            })
            .fail(function(result) {
                error(result);
            });
    }


    function getPath(dirPath, keypath, client) {
        var url = "/local/";
        if (client) {
            url = "/remote/";
        }

        if (!dirPath) {
            r.set("selectKeypath", "");
            $.ajax({
                    type: "get",
                    url: url + "root/",
                    data: JSON.stringify({
                        dirPath: dirPath,
                        client: client,
                    }),
                })
                .done(function(result) {
                    r.set("root", {
                        name: result.data,
                        path: result.data,
                        open: true,
                        client: client,
                    });
                    getPath(result.data, keypath, client);
                })
                .fail(function(result) {
                    error(result);
                });
            return;
        }

        $.ajax({
                type: "get",
                url: url,
                dataType: "json",
                data: JSON.stringify({
                    dirPath: dirPath,
                    client: client,
                }),
            })
            .done(function(result) {
                var files = [];

                for (var i = 0; i < result.data.length; i++) {
                    var f = trimSlash(result.data[i]);
                    var name = f.split("/").pop();
                    if (!client && !r.get("showHidden")) {
                        if (name[0] == ".") {
                            continue;
                        }
                    }

                    files.push({
                        name: name,
                        path: result.data[i],
                        client: client,
                    });

                }
                r.set(keypath, files);
            })
            .fail(function(result) {
                r.set(keypath, {
                    error: result.responseJSON.message,
                });
            });

    }

    function trimSlash(url) {
        if (url.lastIndexOf("/") === url.length - 1) {
            return url.slice(0, url.length - 1);
        }
        return url;
    }


    function error(err) {
        if (typeof err === "string") {
            r.fire("addAlert", "danger", "", err);
            return;
        } else {
            err = err.responseJSON;
            if (err.hasOwnProperty("failures")) {
                for (var i = 0; i < err.failures.length; i++) {
                    r.fire("addAlert", "danger", "", err.failures[i].message);
                }
            } else {
                r.fire("addAlert", "danger", "", err.message);
            }
        }
    }

});
