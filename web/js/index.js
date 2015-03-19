// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

$(document).ready(function() {
    var r = new Ractive({
        el: "main",
        template: "#tMain",
        data: {
            alerts: [],
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
            r.set("modalType", "new");
            $("#profileModal").modal("show");
        },
        "saveNewProfile": function(event) {

        },
        "profileSubmit": function(event) {
            event.original.preventDefault();
        }

    });


    function Profile() {
        this.id = "";
        this.name = "New Profile";
        this.direction = 0;
        this.conflictResolution = 0;
        this.conflictDuration = 0;
        this.active = true;
        this.ignore = [];
        this.localPath = "/home/tshannon/";
        this.remotePath = "/v1/file/testing";
        this.client = {
            url: "",
            user: "",
            password: "",
        };
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
                data: {
                    page: page,
                    type: type,
                }
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
