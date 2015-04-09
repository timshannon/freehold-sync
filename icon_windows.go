// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

func getIconData() []byte {
	data, err := Asset("web/trayIcon.ico")
	if err != nil {
		panic("Windows tray icon asset not found!")
	}
	return data
}
