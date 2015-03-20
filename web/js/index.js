// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

$(document).ready(function() {
    Ractive.DEBUG = true;
    var r = new Ractive({
        el: "main",
        template: "#tMain",
        data: {
            alerts: [],
            page: "main",
        }
    });


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

        },
        "loadProfiles": function() {
            loadProfile();
        },
        "newProfile": function() {
            r.set("currentProfile", new Profile());
            r.set("page", "newProfile");
        },
        "cancelProfile": function() {
            r.set("page", "main");
        },

        "saveNewProfile": function(event) {

        },
        "profileSubmit": function(event) {
            event.original.preventDefault();
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
            getPath("", "root.children");
        },
        "showRemoteModal": function(event) {
            $("#remoteModal").modal("show");
        },
        "treeopen": function(event) {
            r.toggle(event.keypath + ".open");
            getPath(event.context.path, event.keypath + ".children", event.context.client);
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
    });

    r.observe({
        "showHidden": function(newvalue, oldvalue, keypath) {
            getPath("", "root.children");
        },
    });


    function Profile() {
        this.id = "";
        this.name = "";
        this.direction = 0;
        this.conflictResolution = 0;
        this.conflictDuration = 0;
        this.active = true;
        this.ignore = [];
        this.localPath = "";
        this.remotePath = "";
        this.client = new Client();
        //methods
        this.saveNew = function() {
            $.ajax({
                type: "POST",
                url: "/profile/",
                dataType: "json",
                data: JSON.stringify(this),
            });
        };
    }

    function Client() {
        return {
            url: "",
            user: "",
            password: "",
            getToken: function() {

            }(this.bind),
        };
    }


    function loadProfile() {
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

    function loadLogs(page, type) {
        $.ajax({
                type: "get",
                url: "/log/",
                dataType: "json",
                data: JSON.stringify({
                    page: page,
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
                })
                .done(function(result) {
                    r.set("root", {
                        name: result.data,
                        path: result.data,
                        open: true,
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
                    var f = result.data[i];
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
                error(result);
            });

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
